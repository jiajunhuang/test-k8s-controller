package controller

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/time/rate"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	clientset "github.com/jiajunhuang/test/pkg/generated/clientset/versioned"
	"github.com/jiajunhuang/test/pkg/generated/clientset/versioned/scheme"
	webappscheme "github.com/jiajunhuang/test/pkg/generated/clientset/versioned/scheme"
	informers "github.com/jiajunhuang/test/pkg/generated/informers/externalversions/jiajunhuang.com/v1"
	v1lister "github.com/jiajunhuang/test/pkg/generated/listers/jiajunhuang.com/v1"
)

type Controller struct {
	kubeclientset kubernetes.Interface
	webclientset  clientset.Interface

	webappsLister v1lister.WebAppLister
	webappsSynced cache.InformerSynced

	workqueue workqueue.TypedRateLimitingInterface[cache.ObjectName]
	recorder  record.EventRecorder
}

func NewController(
	ctx context.Context,
	kubeclientset kubernetes.Interface,
	webclientset clientset.Interface,
	webappsInformer informers.WebAppInformer,
) (*Controller, error) {
	logger := klog.FromContext(ctx)
	utilruntime.Must(webappscheme.AddToScheme(scheme.Scheme))

	eventBroadcaster := record.NewBroadcaster(record.WithContext(ctx))
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "web-app-controller"})
	ratelimiter := workqueue.NewTypedMaxOfRateLimiter(
		workqueue.NewTypedItemExponentialFailureRateLimiter[cache.ObjectName](5*time.Millisecond, 1000*time.Second),
		&workqueue.TypedBucketRateLimiter[cache.ObjectName]{Limiter: rate.NewLimiter(rate.Limit(50), 300)},
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
	})

	return controller, nil
}

func (c *Controller) enqueueWebApp(obj interface{}) {
	if objectRef, err := cache.ObjectToName(obj); err != nil {
		utilruntime.HandleError(err)
		return
	} else {
		c.workqueue.Add(objectRef)
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

func (c *Controller) syncHandler(ctx context.Context, objectRef cache.ObjectName) error {
	logger := klog.FromContext(ctx)

	webapp, err := c.webappsLister.WebApps(objectRef.Namespace).Get(objectRef.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("webapp '%s' in work queue no longer exists", objectRef.Namespace+"/"+objectRef.Name))
			return nil
		}
		return err
	}

	logger.Info("Syncing webapp", "webapp", webapp, "spec", webapp.Spec)
	return nil
}
