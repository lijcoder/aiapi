package handler

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/model"
)

// ===== 请求/响应结构 =====

type modelItem struct {
	ID                  int64     `json:"id"`
	Provider            string    `json:"provider"`
	Model               string    `json:"model"`
	InputCacheHitPrice  float64   `json:"input_cache_hit_price"`
	InputCacheMissPrice float64   `json:"input_cache_miss_price"`
	OutputPrice         float64   `json:"output_price"`
	MaxContextTokens    int       `json:"max_context_tokens"`
	MaxCompletionTokens int       `json:"max_completion_tokens"`
	CreatedAt           time.Time `json:"created_at"`
}

type ListModelsAdminReq struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type ListModelsAdminResp struct {
	Models []modelItem `json:"models"`
}

type CreateModelReq struct {
	Provider            string  `json:"provider"`
	Model               string  `json:"model"`
	InputCacheHitPrice  float64 `json:"input_cache_hit_price"`
	InputCacheMissPrice float64 `json:"input_cache_miss_price"`
	OutputPrice         float64 `json:"output_price"`
	MaxContextTokens    int     `json:"max_context_tokens"`
	MaxCompletionTokens int     `json:"max_completion_tokens"`
}

type UpdateModelReq struct {
	ID                  int64   `json:"id"`
	InputCacheHitPrice  float64 `json:"input_cache_hit_price"`
	InputCacheMissPrice float64 `json:"input_cache_miss_price"`
	OutputPrice         float64 `json:"output_price"`
	MaxContextTokens    int     `json:"max_context_tokens"`
	MaxCompletionTokens int     `json:"max_completion_tokens"`
}

type ModelIdReq struct {
	ID int64 `json:"id"`
}

// ===== Handler =====

// ListModels 普通用户查询全部模型列表
func ListModels(ctx context.Context) ([]model.Model, *base.BizError) {
	models, err := store.C().Model().List("", "")
	if err != nil {
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	return models, nil
}

// ListModelsAdmin 管理员查询全部模型定价，支持按 provider/model 模糊搜索
func ListModelsAdmin(ctx context.Context, req *ListModelsAdminReq) (*ListModelsAdminResp, *base.BizError) {
	list, err := store.C().Model().List(strings.TrimSpace(req.Provider), strings.TrimSpace(req.Model))
	if err != nil {
		slog.Error("[Model] List failed", "err", err)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	items := make([]modelItem, 0, len(list))
	for _, m := range list {
		items = append(items, toModelItem(m))
	}
	return &ListModelsAdminResp{Models: items}, nil
}

// CreateModel 管理员新增模型定价
func CreateModel(ctx context.Context, req *CreateModelReq) (*modelItem, *base.BizError) {
	if req.Provider == "" || req.Model == "" {
		return nil, base.NewBizError(base.CodeInvalidParams, "provider and model are required")
	}
	// 检查唯一性
	exist, err := store.C().Model().Get(req.Provider, req.Model)
	if err != nil {
		slog.Error("[Model] Get failed", "err", err)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	if exist != nil {
		return nil, base.NewBizError(base.CodeModelExists, "model already exists")
	}

	m := &model.Model{
		Provider:            req.Provider,
		Model:               req.Model,
		InputCacheHitPrice:  req.InputCacheHitPrice,
		InputCacheMissPrice: req.InputCacheMissPrice,
		OutputPrice:         req.OutputPrice,
		MaxContextTokens:    req.MaxContextTokens,
		MaxCompletionTokens: req.MaxCompletionTokens,
	}
	if err := store.C().Model().Create(m); err != nil {
		if store.IsUniqueConstraintErr(err) {
			return nil, base.NewBizError(base.CodeModelExists, "model already exists")
		}
		slog.Error("[Model] Create failed", "err", err)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	item := toModelItem(*m)
	return &item, nil
}

// UpdateModel 管理员编辑模型定价（provider+model 不可改）
func UpdateModel(ctx context.Context, req *UpdateModelReq) (*modelItem, *base.BizError) {
	if req.ID <= 0 {
		return nil, base.NewBizError(base.CodeInvalidParams, "id is required")
	}
	m, err := store.C().Model().GetByID(req.ID)
	if err != nil {
		slog.Error("[Model] GetByID failed", "err", err, "id", req.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	if m == nil {
		return nil, base.NewBizError(base.CodeModelNotFound, "model not found")
	}

	m.InputCacheHitPrice = req.InputCacheHitPrice
	m.InputCacheMissPrice = req.InputCacheMissPrice
	m.OutputPrice = req.OutputPrice
	m.MaxContextTokens = req.MaxContextTokens
	m.MaxCompletionTokens = req.MaxCompletionTokens
	if err := store.C().Model().Update(m); err != nil {
		slog.Error("[Model] Update failed", "err", err, "id", req.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	item := toModelItem(*m)
	return &item, nil
}

// DeleteModel 管理员删除模型定价
func DeleteModel(ctx context.Context, req *ModelIdReq) (*struct{}, *base.BizError) {
	if req.ID <= 0 {
		return nil, base.NewBizError(base.CodeInvalidParams, "id is required")
	}
	m, err := store.C().Model().GetByID(req.ID)
	if err != nil {
		slog.Error("[Model] GetByID failed", "err", err, "id", req.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	if m == nil {
		return nil, base.NewBizError(base.CodeModelNotFound, "model not found")
	}
	if err := store.C().Model().Delete(req.ID); err != nil {
		slog.Error("[Model] Delete failed", "err", err, "id", req.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	return &struct{}{}, nil
}

// ===== 工具函数 =====

func toModelItem(m model.Model) modelItem {
	return modelItem{
		ID:                  m.ID,
		Provider:            m.Provider,
		Model:               m.Model,
		InputCacheHitPrice:  m.InputCacheHitPrice,
		InputCacheMissPrice: m.InputCacheMissPrice,
		OutputPrice:         m.OutputPrice,
		MaxContextTokens:    m.MaxContextTokens,
		MaxCompletionTokens: m.MaxCompletionTokens,
		CreatedAt:           m.CreatedAt,
	}
}
