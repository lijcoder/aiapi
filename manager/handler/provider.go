package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/service"
	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/model"
)

// ===== 请求/响应结构 =====

// providerConfig 脱敏后的配置展示（用于列表/查看）
type providerConfig struct {
	Domain  string              `json:"domain"`
	Headers map[string][]string `json:"headers"`
}

type providerItem struct {
	Type      string         `json:"type"`
	Config    providerConfig `json:"config"`
	Enabled   bool           `json:"enabled"`
	CreatedAt time.Time      `json:"created_at"`
}

// ListProvidersReq 提供商列表查询请求（分页）
type ListProvidersReq struct {
	base.PageReq
}

type CreateProviderReq struct {
	Type    string             `json:"type"`
	Domain  string             `json:"domain"`
	Headers map[string][]string `json:"headers"`
}

type UpdateProviderReq struct {
	Type    string             `json:"type"`
	Domain  string             `json:"domain"`
	Headers map[string][]string `json:"headers"`
}

type ToggleProviderReq struct {
	Type string `json:"type"`
}

// ===== Handler =====

// ListProviders 列出全部 Provider
func ListProviders(ctx context.Context, req *ListProvidersReq) (*base.PageResult[providerItem], *base.BizError) {
	pc := &store.PageContext{Page: req.Page, PageSize: req.PageSize}
	list, err := store.C().SetPage(pc).Provider().List()
	if err != nil {
		slog.Error("[Provider] List failed", "err", err)
		return nil, base.ErrInternal
	}
	items := make([]providerItem, 0, len(list))
	for _, p := range list {
		// 解密+解析统一走 service；配置损坏的行降级为空配置展示，不拖垮整页
		cfg, err := service.ParseProviderConfig(&p)
		if err != nil {
			slog.Error("[Provider] parse config failed", "err", err, "type", p.Type)
		}
		if cfg == nil {
			cfg = &model.ProviderConfig{}
		}
		items = append(items, providerItem{
			Type:      p.Type,
			Config:    providerConfig{Domain: cfg.Domain, Headers: cfg.Headers},
			Enabled:   p.Enabled,
			CreatedAt: p.CreatedAt,
		})
	}
	return &base.PageResult[providerItem]{Items: items, Total: pc.Total, Page: pc.Page, PageSize: pc.PageSize}, nil
}

// CreateProvider 新增 Provider
func CreateProvider(ctx context.Context, req *CreateProviderReq) (*providerItem, *base.BizError) {
	if req.Type == "" {
		return nil, base.ErrBadReq("类型不能为空")
	}
	// 检查唯一性
	exist, err := store.C().Provider().GetByType(req.Type)
	if err != nil {
		slog.Error("[Provider] GetByType failed", "err", err)
		return nil, base.ErrInternal
	}
	if exist != nil {
		return nil, base.ErrBadReq("Provider 类型已存在")
	}

	cfgJSON, err := json.Marshal(model.ProviderConfig{Domain: req.Domain, Headers: req.Headers})
	if err != nil {
		slog.Error("[Provider] marshal config failed", "err", err)
		return nil, base.ErrInternal
	}
	p := &model.Provider{Type: req.Type, Config: string(cfgJSON), Enabled: true}
	// 敏感字段（config 含上游 Key）加密后落库
	if err := service.EncryptProvider(p); err != nil {
		slog.Error("[Provider] encrypt config failed", "err", err)
		return nil, base.ErrInternal
	}
	if err := store.C().Provider().Create(p); err != nil {
		if store.IsUniqueConstraintErr(err) {
			return nil, base.ErrBadReq("Provider 类型已存在")
		}
		slog.Error("[Provider] Create failed", "err", err)
		return nil, base.ErrInternal
	}

	return &providerItem{
		Type:    p.Type,
		Config:  providerConfig{Domain: req.Domain, Headers: req.Headers},
		Enabled: true,
	}, nil
}

// UpdateProvider 编辑 Provider（type 不可改）
func UpdateProvider(ctx context.Context, req *UpdateProviderReq) (*providerItem, *base.BizError) {
	if req.Type == "" {
		return nil, base.ErrBadReq("类型不能为空")
	}
	p, err := store.C().Provider().GetByType(req.Type)
	if err != nil {
		slog.Error("[Provider] GetByType failed", "err", err)
		return nil, base.ErrInternal
	}
	if p == nil {
		return nil, base.ErrNotFound("Provider 不存在")
	}

	cfgJSON, err := json.Marshal(model.ProviderConfig{Domain: req.Domain, Headers: req.Headers})
	if err != nil {
		slog.Error("[Provider] marshal config failed", "err", err)
		return nil, base.ErrInternal
	}
	p.Config = string(cfgJSON)
	// 敏感字段（config 含上游 Key）加密后落库
	if err := service.EncryptProvider(p); err != nil {
		slog.Error("[Provider] encrypt config failed", "err", err)
		return nil, base.ErrInternal
	}
	if err := store.C().Provider().Update(p); err != nil {
		slog.Error("[Provider] Update failed", "err", err)
		return nil, base.ErrInternal
	}

	return &providerItem{
		Type:    p.Type,
		Config:  providerConfig{Domain: req.Domain, Headers: req.Headers},
		Enabled: p.Enabled,
	}, nil
}

// ToggleProvider 启用/禁用 Provider
func ToggleProvider(ctx context.Context, req *ToggleProviderReq) (*struct{}, *base.BizError) {
	if req.Type == "" {
		return nil, base.ErrBadReq("类型不能为空")
	}
	p, err := store.C().Provider().GetByType(req.Type)
	if err != nil {
		slog.Error("[Provider] GetByType failed", "err", err)
		return nil, base.ErrInternal
	}
	if p == nil {
		return nil, base.ErrNotFound("Provider 不存在")
	}
	if err := store.C().Provider().SetEnabled(req.Type, !p.Enabled); err != nil {
		slog.Error("[Provider] SetEnabled failed", "err", err, "type", req.Type)
		return nil, base.ErrInternal
	}
	return &struct{}{}, nil
}

