package tasks

import (
	"context"

	v1 "github.com/jiajunhuang/test/pkg/apis/jiajunhuang.com/v1"
)

// TaskExecutor 定义任务执行器接口
type TaskExecutor interface {
	Name() string
	PreCreate(ctx context.Context, webapp *v1.WebApp) error
	Create(ctx context.Context, webapp *v1.WebApp) error
	PostCreate(ctx context.Context, webapp *v1.WebApp) error
	PreDelete(ctx context.Context, webapp *v1.WebApp) error
	Delete(ctx context.Context, webapp *v1.WebApp) error
	PostDelete(ctx context.Context, webapp *v1.WebApp) error
}
