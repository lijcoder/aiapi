package service

import (
	"errors"
	"strings"

	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/model"
)

var ErrModelAlreadyExists = errors.New("model already exists")

type pricingConfigError struct{ err error }

func (e *pricingConfigError) Error() string { return e.err.Error() }
func (e *pricingConfigError) Unwrap() error { return e.err }

// ModelService 封装模型相关业务逻辑。
type ModelService struct{}

// NewModelService 创建 ModelService。
func NewModelService() *ModelService { return &ModelService{} }

// UpstreamModelName 返回转发时发往上游的模型名：未配置 provider_model（空或纯空格）时
// 回退到 model。对用户可见的 model 是鉴权/计价/白名单的口径，provider_model 只影响转发。
// 回退同时兼容尚未回填 provider_model 的存量库。
func UpstreamModelName(m *model.Model) string {
	if m == nil {
		return ""
	}
	if name := strings.TrimSpace(m.ProviderModel); name != "" {
		return name
	}
	return m.Model
}

// Create 创建模型并校验、规范化其计费配置。
func (s *ModelService) Create(m *model.Model) error {
	// 未传提供商模型名时与 model 保持一致，保证库中不留空
	m.ProviderModel = UpstreamModelName(m)
	pricingConfig, err := NormalizePricingConfig(m.PricingConfig)
	if err != nil {
		return &pricingConfigError{err: err}
	}
	m.PricingConfig = pricingConfig
	existing, err := store.C().Model().Get(m.Provider, m.Model)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrModelAlreadyExists
	}
	if err := store.C().Model().Create(m); err != nil {
		if store.IsUniqueConstraintErr(err) {
			return ErrModelAlreadyExists
		}
		return err
	}
	return nil
}

// Update 更新模型的可编辑配置，并校验、规范化其计费规则。
func (s *ModelService) Update(m *model.Model) error {
	// 提供商模型名传空表示「跟随模型名」，同样归一化后落库
	m.ProviderModel = UpstreamModelName(m)
	pricingConfig, err := NormalizePricingConfig(m.PricingConfig)
	if err != nil {
		return &pricingConfigError{err: err}
	}
	m.PricingConfig = pricingConfig
	return store.C().Model().Update(m)
}

// ListAvailableModels 查询某 API Key 在指定 provider 下可用的模型列表。
// 口径与 proxy 鉴权一致：模型必须配置在该 provider 下，且 Key 为 whitelist 策略时
// 只返回白名单内的模型；all（或未配置）策略返回该 provider 全量模型。
func (s *ModelService) ListAvailableModels(provider string, apiKeyID int64) ([]model.Model, error) {
	policy, modelIDs, err := NewApiKeyService().GetModelAccess(apiKeyID)
	if err != nil {
		return nil, err
	}
	if policy != store.ModelPolicyWhitelist {
		return store.C().Model().ListByProvider(provider)
	}
	if len(modelIDs) == 0 {
		return nil, nil
	}
	models, err := store.C().Model().ListByIDs(modelIDs)
	if err != nil {
		return nil, err
	}
	// 白名单按 model_id 全局配置，列表只保留当前 provider 下的条目
	available := make([]model.Model, 0, len(models))
	for _, m := range models {
		if m.Provider == provider {
			available = append(available, m)
		}
	}
	return available, nil
}

// IsPricingConfigError 报告错误是否来自用户提供的计费配置。
func IsPricingConfigError(err error) bool {
	var target *pricingConfigError
	return errors.As(err, &target)
}
