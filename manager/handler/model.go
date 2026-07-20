package handler

import (
	"context"

	"github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/model"
)

// ListModels 查询全部模型列表
func ListModels(ctx context.Context) ([]model.Model, *base.BizError) {
	models, err := store.C().Model().List()
	if err != nil {
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	return models, nil
}
