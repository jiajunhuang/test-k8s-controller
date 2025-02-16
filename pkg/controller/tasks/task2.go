package tasks

import (
	"context"

	v1 "github.com/jiajunhuang/test/pkg/apis/jiajunhuang.com/v1"
	"k8s.io/klog/v2"
)

type Step2TaskExecutor struct {
	// 可以在这里添加需要的依赖
}

func (e *Step2TaskExecutor) Name() string {
	return "step2"
}

// 实现所有接口方法
func (e *Step2TaskExecutor) PreCreate(ctx context.Context, webapp *v1.WebApp) error {
	klog.FromContext(ctx).Info("执行 PreCreate", "webapp", webapp.Name, "step", e.Name())
	return nil
}

func (e *Step2TaskExecutor) Create(ctx context.Context, webapp *v1.WebApp) error {
	klog.FromContext(ctx).Info("执行 Create", "webapp", webapp.Name, "step", e.Name())
	return nil
}

func (e *Step2TaskExecutor) PostCreate(ctx context.Context, webapp *v1.WebApp) error {
	klog.FromContext(ctx).Info("执行 PostCreate", "webapp", webapp.Name, "step", e.Name())
	return nil
}

func (e *Step2TaskExecutor) PreDelete(ctx context.Context, webapp *v1.WebApp) error {
	klog.FromContext(ctx).Info("执行 PreDelete", "webapp", webapp.Name, "step", e.Name())
	return nil
}

func (e *Step2TaskExecutor) Delete(ctx context.Context, webapp *v1.WebApp) error {
	klog.FromContext(ctx).Info("执行 Delete", "webapp", webapp.Name, "step", e.Name())
	return nil
}

func (e *Step2TaskExecutor) PostDelete(ctx context.Context, webapp *v1.WebApp) error {
	klog.FromContext(ctx).Info("执行 PostDelete", "webapp", webapp.Name, "step", e.Name())
	return nil
}
