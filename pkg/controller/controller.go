package controller

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/time/rate"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	v1 "github.com/jiajunhuang/test/pkg/apis/jiajunhuang.com/v1"
	clientset "github.com/jiajunhuang/test/pkg/generated/clientset/versioned"
	webappscheme "github.com/jiajunhuang/test/pkg/generated/clientset/versioned/scheme"
	informers "github.com/jiajunhuang/test/pkg/generated/informers/externalversions/jiajunhuang.com/v1"
	v1lister "github.com/jiajunhuang/test/pkg/generated/listers/jiajunhuang.com/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TaskExecutor 定义任务执行器接口
type TaskExecutor interface {
	PreCreate(ctx context.Context, webapp *v1.WebApp) error
	Create(ctx context.Context, webapp *v1.WebApp) error
	PostCreate(ctx context.Context, webapp *v1.WebApp) error
	PreDelete(ctx context.Context, webapp *v1.WebApp) error
	Delete(ctx context.Context, webapp *v1.WebApp) error
	PostDelete(ctx context.Context, webapp *v1.WebApp) error
}

// defaultTaskExecutor 实现 TaskExecutor 接口
type defaultTaskExecutor struct {
	// 可以在这里添加需要的依赖
}

// 实现所有接口方法
func (e *defaultTaskExecutor) PreCreate(ctx context.Context, webapp *v1.WebApp) error {
	klog.FromContext(ctx).Info("执行 PreCreate", "webapp", webapp.Name)
	return nil
}

func (e *defaultTaskExecutor) Create(ctx context.Context, webapp *v1.WebApp) error {
	klog.FromContext(ctx).Info("执行 Create", "webapp", webapp.Name)
	return nil
}

func (e *defaultTaskExecutor) PostCreate(ctx context.Context, webapp *v1.WebApp) error {
	klog.FromContext(ctx).Info("执行 PostCreate", "webapp", webapp.Name)
	return nil
}

func (e *defaultTaskExecutor) PreDelete(ctx context.Context, webapp *v1.WebApp) error {
	klog.FromContext(ctx).Info("执行 PreDelete", "webapp", webapp.Name)
	return nil
}

func (e *defaultTaskExecutor) Delete(ctx context.Context, webapp *v1.WebApp) error {
	klog.FromContext(ctx).Info("执行 Delete", "webapp", webapp.Name)
	return nil
}

func (e *defaultTaskExecutor) PostDelete(ctx context.Context, webapp *v1.WebApp) error {
	klog.FromContext(ctx).Info("执行 PostDelete", "webapp", webapp.Name)
	return nil
}

type Controller struct {
	kubeclientset kubernetes.Interface
	webclientset  clientset.Interface

	webappsLister v1lister.WebAppLister
	webappsSynced cache.InformerSynced

	workqueue workqueue.TypedRateLimitingInterface[types.NamespacedName]
	recorder  record.EventRecorder

	executor TaskExecutor
}

const (
	// 添加 finalizer 常量
	webappFinalizer = "webapp.jiajunhuang.com/finalizer"
)

func NewController(
	ctx context.Context,
	kubeclientset kubernetes.Interface,
	webclientset clientset.Interface,
	webappsInformer informers.WebAppInformer,
) (*Controller, error) {
	logger := klog.FromContext(ctx)
	utilruntime.Must(webappscheme.AddToScheme(webappscheme.Scheme))

	eventBroadcaster := record.NewBroadcaster(record.WithContext(ctx))
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(webappscheme.Scheme, corev1.EventSource{Component: "web-app-controller"})
	ratelimiter := workqueue.NewTypedMaxOfRateLimiter(
		workqueue.NewTypedItemExponentialFailureRateLimiter[types.NamespacedName](5*time.Millisecond, 1000*time.Second),
		&workqueue.TypedBucketRateLimiter[types.NamespacedName]{Limiter: rate.NewLimiter(rate.Limit(50), 300)},
	)

	controller := &Controller{
		kubeclientset: kubeclientset,
		webclientset:  webclientset,
		webappsLister: webappsInformer.Lister(),
		webappsSynced: webappsInformer.Informer().HasSynced,
		workqueue:     workqueue.NewTypedRateLimitingQueue(ratelimiter),
		recorder:      recorder,
	}

	logger.Info("Starting web-app-controller")

	webappsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueWebApp,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueWebApp(new)
		},
		DeleteFunc: controller.enqueueWebApp,
	})

	return controller, nil
}

func (c *Controller) enqueueWebApp(obj interface{}) {
	if objectRef, err := cache.ObjectToName(obj); err != nil {
		utilruntime.HandleError(err)
		return
	} else {
		c.workqueue.Add(types.NamespacedName{Namespace: objectRef.Namespace, Name: objectRef.Name})
	}
}

func (c *Controller) Run(ctx context.Context) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	logger := klog.FromContext(ctx)
	logger.Info("Starting web-app-controller")

	if ok := cache.WaitForCacheSync(ctx.Done(), c.webappsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	logger.Info("Starting workers")

	go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	<-ctx.Done()
	return nil
}

func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

func (c *Controller) processNextWorkItem(ctx context.Context) bool {
	key, quit := c.workqueue.Get()
	if quit {
		return false
	}
	defer c.workqueue.Done(key)

	err := c.syncHandler(ctx, key)
	if err == nil {
		c.workqueue.Forget(key)
		return true
	}

	utilruntime.HandleError(fmt.Errorf("error syncing '%s': %s", key, err))
	c.workqueue.AddRateLimited(key)
	return true
}

func (c *Controller) syncHandler(ctx context.Context, objectRef types.NamespacedName) error {
	logger := klog.FromContext(ctx)
	executor := &defaultTaskExecutor{}

	webapp, err := c.webappsLister.WebApps(objectRef.Namespace).Get(objectRef.Name)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	// 如果对象不存在，直接返回
	if apierrors.IsNotFound(err) {
		logger.Info("WebApp 已经被删除", "webapp", objectRef.String())
		return nil
	}

	// 处理删除操作
	if webapp.DeletionTimestamp != nil {
		return c.handleDeletion(ctx, webapp, executor)
	}

	// 处理创建/更新操作
	return c.handleCreation(ctx, webapp, executor)
}

// 处理删除操作
func (c *Controller) handleDeletion(ctx context.Context, webapp *v1.WebApp, executor TaskExecutor) error {
	logger := klog.FromContext(ctx)

	// 检查是否包含我们的 finalizer
	if !containsFinalizer(webapp.Finalizers, webappFinalizer) {
		return nil
	}

	// 执行删除相关操作
	if err := executor.PreDelete(ctx, webapp); err != nil {
		return fmt.Errorf("执行 PreDelete 失败: %v", err)
	}
	if err := executor.Delete(ctx, webapp); err != nil {
		return fmt.Errorf("执行 Delete 失败: %v", err)
	}
	if err := executor.PostDelete(ctx, webapp); err != nil {
		return fmt.Errorf("执行 PostDelete 失败: %v", err)
	}

	// 删除 finalizer
	webappCopy := webapp.DeepCopy()
	webappCopy.Finalizers = removeFinalizer(webappCopy.Finalizers, webappFinalizer)
	_, err := c.webclientset.JiajunhuangV1().WebApps(webapp.Namespace).Update(ctx, webappCopy, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("移除 finalizer 失败: %v", err)
	}

	logger.Info("成功完成删除操作", "webapp", webapp.Name)
	return nil
}

// 处理创建/更新操作
func (c *Controller) handleCreation(ctx context.Context, webapp *v1.WebApp, executor TaskExecutor) error {
	// 确保有 finalizer
	if !containsFinalizer(webapp.Finalizers, webappFinalizer) {
		webappCopy := webapp.DeepCopy()
		webappCopy.Finalizers = append(webappCopy.Finalizers, webappFinalizer)
		var err error
		webapp, err = c.webclientset.JiajunhuangV1().WebApps(webapp.Namespace).Update(ctx, webappCopy, metav1.UpdateOptions{})
		if err != nil {
			// 如果是验证错误，记录详细信息并返回
			if apierrors.IsInvalid(err) {
				logger := klog.FromContext(ctx)
				logger.Error(err, "资源验证失败",
					"webapp", klog.KObj(webapp),
					"details", err.(*apierrors.StatusError).ErrStatus.Details)
				// 这种情况下不需要重试
				return nil
			}
			return fmt.Errorf("添加 finalizer 失败: %v", err)
		}
	}

	if err := executor.PreCreate(ctx, webapp); err != nil {
		return fmt.Errorf("执行 PreCreate 失败: %v", err)
	}
	if err := executor.Create(ctx, webapp); err != nil {
		return fmt.Errorf("执行 Create 失败: %v", err)
	}
	if err := executor.PostCreate(ctx, webapp); err != nil {
		return fmt.Errorf("执行 PostCreate 失败: %v", err)
	}

	return nil
}

// 辅助函数：检查是否包含 finalizer
func containsFinalizer(finalizers []string, finalizer string) bool {
	for _, f := range finalizers {
		if f == finalizer {
			return true
		}
	}
	return false
}

// 辅助函数：移除 finalizer
func removeFinalizer(finalizers []string, finalizer string) []string {
	var result []string
	for _, f := range finalizers {
		if f != finalizer {
			result = append(result, f)
		}
	}
	return result
}
