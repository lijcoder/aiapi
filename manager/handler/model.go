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
	SupportsText        bool      `json:"supports_text"`
	SupportsImage       bool      `json:"supports_image"`
	SupportsVideo       bool      `json:"supports_video"`
	CreatedAt           time.Time `json:"created_at"`
}

type ListModelsAdminReq struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	base.PageReq
}

type CreateModelReq struct {
	Provider            string  `json:"provider"`
	Model               string  `json:"model"`
	InputCacheHitPrice  float64 `json:"input_cache_hit_price"`
	InputCacheMissPrice float64 `json:"input_cache_miss_price"`
	OutputPrice         float64 `json:"output_price"`
	MaxContextTokens    int     `json:"max_context_tokens"`
	MaxCompletionTokens int     `json:"max_completion_tokens"`
	SupportsText        bool    `json:"supports_text"`
	SupportsImage       bool    `json:"supports_image"`
	SupportsVideo       bool    `json:"supports_video"`
}

type UpdateModelReq struct {
	ID                  int64   `json:"id"`
	InputCacheHitPrice  float64 `json:"input_cache_hit_price"`
	InputCacheMissPrice float64 `json:"input_cache_miss_price"`
	OutputPrice         float64 `json:"output_price"`
	MaxContextTokens    int     `json:"max_context_tokens"`
	MaxCompletionTokens int     `json:"max_completion_tokens"`
	SupportsText        bool    `json:"supports_text"`
	SupportsImage       bool    `json:"supports_image"`
	SupportsVideo       bool    `json:"supports_video"`
}

type ModelIdReq struct {
	ID int64 `json:"id"`
}

// ===== Handler =====

// ListModelsReq 普通用户模型列表查询请求（支持搜索 + 分页）
type ListModelsReq struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	base.PageReq
}

// ListModels 普通用户查询模型列表，支持按 provider/model 模糊搜索
func ListModels(ctx context.Context, req *ListModelsReq) (*base.PageResult[model.Model], *base.BizError) {
	pc := &store.PageContext{Page: req.Page, PageSize: req.PageSize}
	list, err := store.C().SetPage(pc).Model().List(strings.TrimSpace(req.Provider), strings.TrimSpace(req.Model))
	if err != nil {
		slog.Error("[Model] List failed", "err", err)
		return nil, base.ErrInternal
	}
	return &base.PageResult[model.Model]{Items: list, Total: pc.Total, Page: pc.Page, PageSize: pc.PageSize}, nil
}

// ListModelsAdmin 管理员查询全部模型，支持按 provider/model 模糊搜索
func ListModelsAdmin(ctx context.Context, req *ListModelsAdminReq) (*base.PageResult[modelItem], *base.BizError) {
	pc := &store.PageContext{Page: req.Page, PageSize: req.PageSize}
	list, err := store.C().SetPage(pc).Model().List(strings.TrimSpace(req.Provider), strings.TrimSpace(req.Model))
	if err != nil {
		slog.Error("[Model] List failed", "err", err)
		return nil, base.ErrInternal
	}
	items := make([]modelItem, 0, len(list))
	for _, m := range list {
		items = append(items, toModelItem(m))
	}
	return &base.PageResult[modelItem]{Items: items, Total: pc.Total, Page: pc.Page, PageSize: pc.PageSize}, nil
}

// CreateModel 管理员新增模型
func CreateModel(ctx context.Context, req *CreateModelReq) (*modelItem, *base.BizError) {
	if req.Provider == "" || req.Model == "" {
		return nil, base.ErrBadReq("provider 和 model 不能为空")
	}
	// 检查唯一性
	exist, err := store.C().Model().Get(req.Provider, req.Model)
	if err != nil {
		slog.Error("[Model] Get failed", "err", err)
		return nil, base.ErrInternal
	}
	if exist != nil {
		return nil, base.ErrBadReq("模型已存在")
	}

	m := &model.Model{
		Provider:            req.Provider,
		Model:               req.Model,
		InputCacheHitPrice:  req.InputCacheHitPrice,
		InputCacheMissPrice: req.InputCacheMissPrice,
		OutputPrice:         req.OutputPrice,
		MaxContextTokens:    req.MaxContextTokens,
		MaxCompletionTokens: req.MaxCompletionTokens,
		SupportsText:        req.SupportsText,
		SupportsImage:       req.SupportsImage,
		SupportsVideo:       req.SupportsVideo,
	}
	if err := store.C().Model().Create(m); err != nil {
		if store.IsUniqueConstraintErr(err) {
			return nil, base.ErrBadReq("模型已存在")
		}
		slog.Error("[Model] Create failed", "err", err)
		return nil, base.ErrInternal
	}
	item := toModelItem(*m)
	return &item, nil
}

// UpdateModel 管理员编辑模型（provider+model 不可改）
func UpdateModel(ctx context.Context, req *UpdateModelReq) (*modelItem, *base.BizError) {
	if req.ID <= 0 {
		return nil, base.ErrBadReq("id 不能为空")
	}
	m, err := store.C().Model().GetByID(req.ID)
	if err != nil {
		slog.Error("[Model] GetByID failed", "err", err, "id", req.ID)
		return nil, base.ErrInternal
	}
	if m == nil {
		return nil, base.ErrNotFound("模型不存在")
	}

	m.InputCacheHitPrice = req.InputCacheHitPrice
	m.InputCacheMissPrice = req.InputCacheMissPrice
	m.OutputPrice = req.OutputPrice
	m.MaxContextTokens = req.MaxContextTokens
	m.MaxCompletionTokens = req.MaxCompletionTokens
	m.SupportsText = req.SupportsText
	m.SupportsImage = req.SupportsImage
	m.SupportsVideo = req.SupportsVideo
	if err := store.C().Model().Update(m); err != nil {
		slog.Error("[Model] Update failed", "err", err, "id", req.ID)
		return nil, base.ErrInternal
	}
	item := toModelItem(*m)
	return &item, nil
}

// DeleteModel 管理员删除模型
func DeleteModel(ctx context.Context, req *ModelIdReq) (*struct{}, *base.BizError) {
	if req.ID <= 0 {
		return nil, base.ErrBadReq("id 不能为空")
	}
	m, err := store.C().Model().GetByID(req.ID)
	if err != nil {
		slog.Error("[Model] GetByID failed", "err", err, "id", req.ID)
		return nil, base.ErrInternal
	}
	if m == nil {
		return nil, base.ErrNotFound("模型不存在")
	}
	if err := store.C().Model().Delete(req.ID); err != nil {
		slog.Error("[Model] Delete failed", "err", err, "id", req.ID)
		return nil, base.ErrInternal
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
		SupportsText:        m.SupportsText,
		SupportsImage:       m.SupportsImage,
		SupportsVideo:       m.SupportsVideo,
		CreatedAt:           m.CreatedAt,
	}
}
