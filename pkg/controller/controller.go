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
	"github.com/jiajunhuang/test/pkg/controller/tasks"
	clientset "github.com/jiajunhuang/test/pkg/generated/clientset/versioned"
	webappscheme "github.com/jiajunhuang/test/pkg/generated/clientset/versioned/scheme"
	informers "github.com/jiajunhuang/test/pkg/generated/informers/externalversions/jiajunhuang.com/v1"
	v1lister "github.com/jiajunhuang/test/pkg/generated/listers/jiajunhuang.com/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Controller struct {
	kubeclientset kubernetes.Interface
	webclientset  clientset.Interface

	webappsLister v1lister.WebAppLister
	webappsSynced cache.InformerSynced

	workqueue workqueue.TypedRateLimitingInterface[types.NamespacedName]
	recorder  record.EventRecorder

	tasks []tasks.TaskExecutor
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
	tasks []tasks.TaskExecutor,
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
		tasks:         tasks,
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
		// execute deletion tasks in reverse order
		for i := len(c.tasks) - 1; i >= 0; i-- {
			if err := c.handleDeletion(ctx, webapp, c.tasks[i]); err != nil {
				return err
			}
		}
		return nil
	}

	// 处理创建/更新操作
	for _, executor := range c.tasks {
		if err := c.handleCreation(ctx, webapp, executor); err != nil {
			return err
		}
	}
	return nil
}

// 处理删除操作
func (c *Controller) handleDeletion(ctx context.Context, webapp *v1.WebApp, executor tasks.TaskExecutor) error {
	logger := klog.FromContext(ctx)

	// 获取最新版本的对象
	currentWebApp, err := c.webclientset.JiajunhuangV1().WebApps(webapp.Namespace).Get(ctx, webapp.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("获取最新版本的 WebApp 失败: %v", err)
	}

	// 检查是否包含我们的 finalizer
	if !containsFinalizer(currentWebApp.Finalizers, webappFinalizer) {
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

	// 只有当这是最后一个任务时才移除 finalizer
	if executor == c.tasks[0] {
		// 使用最新版本的对象移除 finalizer
		webappCopy := currentWebApp.DeepCopy()
		webappCopy.Finalizers = removeFinalizer(webappCopy.Finalizers, webappFinalizer)
		_, err = c.webclientset.JiajunhuangV1().WebApps(webapp.Namespace).Update(ctx, webappCopy, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("移除 finalizer 失败: %v", err)
		}
	}

	logger.Info("成功完成删除操作", "webapp", webapp.Name, "executor", fmt.Sprintf("%T", executor))
	return nil
}

// 处理创建/更新操作
func (c *Controller) handleCreation(ctx context.Context, webapp *v1.WebApp, executor tasks.TaskExecutor) error {
	logger := klog.FromContext(ctx)

	// 添加 finalizer 前先获取最新版本的对象
	currentWebApp, err := c.webclientset.JiajunhuangV1().WebApps(webapp.Namespace).Get(ctx, webapp.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取最新版本的 WebApp 失败: %v", err)
	}

	// 确保有 finalizer
	if !containsFinalizer(currentWebApp.Finalizers, webappFinalizer) {
		webappCopy := currentWebApp.DeepCopy()
		webappCopy.Finalizers = append(webappCopy.Finalizers, webappFinalizer)
		currentWebApp, err = c.webclientset.JiajunhuangV1().WebApps(webapp.Namespace).Update(ctx, webappCopy, metav1.UpdateOptions{})
		if err != nil {
			if apierrors.IsInvalid(err) {
				logger.Error(err, "资源验证失败",
					"webapp", klog.KObj(webapp),
					"details", err.(*apierrors.StatusError).ErrStatus.Details)
				return nil
			}
			return fmt.Errorf("添加 finalizer 失败: %v", err)
		}
		// 更新成功后，使用最新版本继续处理
		webapp = currentWebApp
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
