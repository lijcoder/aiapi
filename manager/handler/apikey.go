package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"

	"github.com/lijcoder/aiapi/manager/base"
	"github.com/lijcoder/aiapi/service"
	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/model"
)

// apiKeyItem API Key 视图对象。列表中 Key 字段已脱敏，明文只在创建时返回。
type apiKeyItem struct {
	ID        int64   `json:"id"`
	UserID    int64   `json:"user_id"`
	Key       string  `json:"key"`
	Name      string  `json:"name"`
	Budget    float64 `json:"budget"`
	Unlimited bool    `json:"unlimited"`
	Enabled   bool    `json:"enabled"`
	CreatedAt string  `json:"created_at"`
}

// CreateApiKeyReq 创建 API Key 请求
type CreateApiKeyReq struct {
	Name      string  `json:"name"`
	Budget    float64 `json:"budget"`
	Unlimited bool    `json:"unlimited"`
}

// CreateApiKeyResp 创建 API Key 响应；仅此接口返回明文 key
type CreateApiKeyResp struct {
	ApiKey apiKeyItem `json:"apiKey"`
}

// ApiKeyIdReq 按 ID 操作 API Key 的请求（启停/删除/改名）
type ApiKeyIdReq struct {
	ID int64 `json:"id"`
}

// ApiKeyListSelfReq 当前用户 API Key 列表查询请求（分页）
type ApiKeyListSelfReq struct {
	base.PageReq
}

// ApiKeyListReq 超管查询指定用户的 API Key 列表请求（分页）
type ApiKeyListReq struct {
	UserID int64 `json:"user_id"`
	base.PageReq
}

// ApiKeyAdminReq 超管管理指定用户的 API Key 请求

type ApiKeyAdminReq struct {
	UserID int64 `json:"user_id"`
}

// RenameApiKeyReq 改名请求
type RenameApiKeyReq struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// UpdateBudgetApiKeyReq 修改额度请求：unimited=false 时 budget 生效，且所有
// “有限额” key 的 budget 之和不能超过用户余额。
type UpdateBudgetApiKeyReq struct {
	ID        int64   `json:"id"`
	Budget    float64 `json:"budget"`
	Unlimited bool    `json:"unlimited"`
}

// keyRandBytes 随机字节数；32 字节 → hex 后 64 字符，256bit 熵
const keyRandBytes = 32

// newApiKey 生成新的明文 API Key，形如 sk-<64 hex>。
// 强度由 crypto/rand 保证，唯一性最终由 uq_api_keys_key 兜底。
func newApiKey() (string, error) {
	b := make([]byte, keyRandBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "sk-" + hex.EncodeToString(b), nil
}

// displayKey 对外展示的脱敏 key：直接返回落库的展示串（sk-abc****xyz）。
func displayKey(k model.ApiKey) string {
	return k.KeyShow
}

// toApiKeyItem 将模型转为视图对象。mask 控制是否对 key 脱敏。
// plainKey 仅创建场景传入（明文只在该响应中返回一次），其余场景传空。
func toApiKeyItem(k model.ApiKey, plainKey string) apiKeyItem {
	key := displayKey(k)
	if plainKey != "" {
		key = plainKey
	}
	created := ""
	if !k.CreatedAt.IsZero() {
		created = k.CreatedAt.Format("2006-01-02 15:04:05")
	}
	return apiKeyItem{
		ID:        k.ID,
		UserID:    k.UserID,
		Key:       key,
		Name:      k.Name,
		Budget:    k.Budget,
		Unlimited: k.Unlimited,
		Enabled:   k.Enabled,
		CreatedAt: created,
	}
}

// createApiKeyRetry 在 unique index 冲突时重试生成。实际碰撞概率极低，3 次足够。
// 调用方应已在前面校验过总额度。
// 返回记录与明文 key（明文不落库，仅用于创建响应返回给用户一次）。
func createApiKeyRetry(userID int64, name string, budget float64, unlimited bool) (*model.ApiKey, string, error) {
	for i := 0; i < 3; i++ {
		plain, err := newApiKey()
		if err != nil {
			return nil, "", err
		}
		enc, err := service.EncryptApiKey(plain)
		if err != nil {
			return nil, "", err
		}
		k := &model.ApiKey{
			UserID:    userID,
			KeyHash:   service.ApiKeyHash(plain),
			KeyEnc:    enc,
			KeyShow:   service.ApiKeyShow(plain),
			Name:      name,
			Budget:    budget,
			Unlimited: unlimited,
			Enabled:   true,
		}
		if err := store.C().ApiKey().Create(k); err != nil {
			// 命中 unique index 才重试；其余错误直接返回
			if store.IsUniqueConstraintErr(err) {
				continue
			}
			return nil, "", err
		}
		return k, plain, nil
	}
	return nil, "", errors.New("api key generation failed after retries")
}

// RevealApiKeyReq 查看 API Key 明文请求
type RevealApiKeyReq struct {
	ID int64 `json:"id"`
}

// RevealApiKeyResp 查看 API Key 明文响应
// 旧版本创建的 key（key_enc 为空）无法还原，返回 revealable=false 提示前端。
type RevealApiKeyResp struct {
	Key        string `json:"key"`
	Revealable bool   `json:"revealable"`
}

// revealApiKey 解密 key_enc 还原明文。key_enc 为空表示旧版本 key，不可还原。
func revealApiKey(k *model.ApiKey) (*RevealApiKeyResp, *base.BizError) {
	if k.KeyEnc == "" {
		return &RevealApiKeyResp{Revealable: false}, nil
	}
	plain, err := service.DecryptApiKey(k.KeyEnc)
	if err != nil {
		slog.Error("decrypt api key failed", "err", err, "id", k.ID)
		return nil, base.ErrInternal
	}
	return &RevealApiKeyResp{Key: plain, Revealable: true}, nil
}

// RevealApiKeySelf 普通用户查看自己 API Key 明文。先校验归属当前用户，防止越权。
func RevealApiKeySelf(ctx context.Context, req *RevealApiKeyReq) (*RevealApiKeyResp, *base.BizError) {
	cur := base.CurrentUser(ctx)
	if req.ID <= 0 {
		return nil, base.ErrBadReq("id 不能为空")
	}
	k, err := store.C().ApiKey().GetByID(req.ID)
	if err != nil {
		slog.Error("get api key failed", "err", err, "id", req.ID)
		return nil, base.ErrInternal
	}
	if k == nil || k.UserID != cur.ID {
		return nil, base.ErrNotFound("API Key 不存在")
	}
	return revealApiKey(k)
}

// RevealApiKeyAdmin 超管查看指定 API Key 明文。
func RevealApiKeyAdmin(ctx context.Context, req *RevealApiKeyReq) (*RevealApiKeyResp, *base.BizError) {
	if req.ID <= 0 {
		return nil, base.ErrBadReq("id 不能为空")
	}
	k, err := store.C().ApiKey().GetByID(req.ID)
	if err != nil {
		slog.Error("get api key failed", "err", err, "id", req.ID)
		return nil, base.ErrInternal
	}
	if k == nil {
		return nil, base.ErrNotFound("API Key 不存在")
	}
	return revealApiKey(k)
}

// checkBudgetWithinUserLimit 校验“有限额 key 的 budget 之和”不超过用户余额。
// excludeID 用于“修改现有 key”场景，表示在统计中排除该 key 本身（用新值替换）。
// 用户为无限用户时跳过校验。返回 nil 表示通过。
func checkBudgetWithinUserLimit(cur *model.User, excludeID int64, newBudget float64, newUnlimited bool) *base.BizError {
	if cur.Unlimited {
		return nil
	}
	sum, err := store.C().ApiKey().SumLimitedBudgetByUser(cur.ID)
	if err != nil {
		slog.Error("sum api key budget failed", "err", err, "user_id", cur.ID)
		return base.ErrInternal
	}
	// 排除当前正在修改的 key（如果它原本是有限额的，需从总和中减掉原值）
	if excludeID > 0 {
		exist, gerr := store.C().ApiKey().GetByID(excludeID)
		if gerr != nil {
			slog.Error("get api key for budget check failed", "err", gerr, "id", excludeID)
			return base.ErrInternal
		}
		if exist != nil && exist.UserID == cur.ID && !exist.Unlimited {
			sum -= exist.Budget
		}
	}
	if !newUnlimited {
		sum += newBudget
	}
	if sum > cur.Budget {
		return base.ErrBadReq("有限额密钥的额度总和不能超过账户余额")
	}
	return nil
}

// ListApiKeySelf 查询当前用户的 API Key 列表（key 脱敏，分页）
func ListApiKeySelf(ctx context.Context, req *ApiKeyListSelfReq) (*base.PageResult[apiKeyItem], *base.BizError) {
	cur := base.CurrentUser(ctx)
	pc := &store.PageContext{Page: req.Page, PageSize: req.PageSize}
	keys, err := store.C().SetPage(pc).ApiKey().ListByUser(cur.ID)
	if err != nil {
		slog.Error("list api keys failed", "err", err, "user_id", cur.ID)
		return nil, base.ErrInternal
	}
	items := make([]apiKeyItem, 0, len(keys))
	for _, k := range keys {
		items = append(items, toApiKeyItem(k, ""))
	}
	return &base.PageResult[apiKeyItem]{Items: items, Total: pc.Total, Page: pc.Page, PageSize: pc.PageSize}, nil
}

// CreateApiKeySelf 创建 API Key，明文 key 仅在本响应中返回一次
func CreateApiKeySelf(ctx context.Context, req *CreateApiKeyReq) (*CreateApiKeyResp, *base.BizError) {
	cur := base.CurrentUser(ctx)
	// 名称必填：key 原文不再落库，名称是识别 key 用途的唯一途径
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, base.ErrBadReq("名称不能为空")
	}
	if len(name) > 64 {
		return nil, base.ErrBadReq("名称过长")
	}
	if req.Budget < 0 {
		return nil, base.ErrBadReq("额度不能为负数")
	}
	// 校验有限额 key 的总额不超过用户余额
	if berr := checkBudgetWithinUserLimit(cur, 0, req.Budget, req.Unlimited); berr != nil {
		return nil, berr
	}

	k, plain, err := createApiKeyRetry(cur.ID, name, req.Budget, req.Unlimited)
	if err != nil {
		slog.Error("create api key failed", "err", err, "user_id", cur.ID)
		return nil, base.ErrInternal
	}
	return &CreateApiKeyResp{ApiKey: toApiKeyItem(*k, plain)}, nil
}

// ToggleApiKeySelf 启用/禁用 API Key。先校验归属当前用户，防止越权。
func ToggleApiKeySelf(ctx context.Context, req *ApiKeyIdReq) (*apiKeyItem, *base.BizError) {
	cur := base.CurrentUser(ctx)
	if req.ID <= 0 {
		return nil, base.ErrBadReq("id 不能为空")
	}
	k, err := store.C().ApiKey().GetByID(req.ID)
	if err != nil {
		slog.Error("get api key failed", "err", err, "id", req.ID)
		return nil, base.ErrInternal
	}
	if k == nil || k.UserID != cur.ID {
		return nil, base.ErrNotFound("API Key 不存在")
	}
	if err := store.C().ApiKey().SetEnabled(k.ID, !k.Enabled); err != nil {
		slog.Error("toggle api key failed", "err", err, "id", k.ID)
		return nil, base.ErrInternal
	}
	k.Enabled = !k.Enabled
	item := toApiKeyItem(*k, "")
	return &item, nil
}

// DeleteApiKeySelf 删除 API Key。先校验归属当前用户，防止越权。
func DeleteApiKeySelf(ctx context.Context, req *ApiKeyIdReq) (*struct{}, *base.BizError) {
	cur := base.CurrentUser(ctx)
	if req.ID <= 0 {
		return nil, base.ErrBadReq("id 不能为空")
	}
	k, err := store.C().ApiKey().GetByID(req.ID)
	if err != nil {
		slog.Error("get api key failed", "err", err, "id", req.ID)
		return nil, base.ErrInternal
	}
	if k == nil || k.UserID != cur.ID {
		return nil, base.ErrNotFound("API Key 不存在")
	}
	if err := service.NewApiKeyService().Delete(k.ID); err != nil {
		slog.Error("delete api key failed", "err", err, "id", k.ID)
		return nil, base.ErrInternal
	}
	return &struct{}{}, nil
}

// RenameApiKeySelf 修改 API Key 名称。先校验归属当前用户，防止越权。
func RenameApiKeySelf(ctx context.Context, req *RenameApiKeyReq) (*apiKeyItem, *base.BizError) {
	cur := base.CurrentUser(ctx)
	if req.ID <= 0 {
		return nil, base.ErrBadReq("id 不能为空")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, base.ErrBadReq("名称不能为空")
	}
	if len(name) > 64 {
		return nil, base.ErrBadReq("名称过长")
	}
	k, err := store.C().ApiKey().GetByID(req.ID)
	if err != nil {
		slog.Error("get api key failed", "err", err, "id", req.ID)
		return nil, base.ErrInternal
	}
	if k == nil || k.UserID != cur.ID {
		return nil, base.ErrNotFound("API Key 不存在")
	}
	if err := store.C().ApiKey().UpdateName(k.ID, name); err != nil {
		slog.Error("rename api key failed", "err", err, "id", k.ID)
		return nil, base.ErrInternal
	}
	k.Name = name
	item := toApiKeyItem(*k, "")
	return &item, nil
}

// UpdateBudgetApiKeySelf 修改 API Key 的额度与限额模式。
// unlimited=false 时 budget 生效，且所有“有限额” key 的 budget 之和不能超过用户余额。
// 先校验归属当前用户，防止越权。
func UpdateBudgetApiKeySelf(ctx context.Context, req *UpdateBudgetApiKeyReq) (*apiKeyItem, *base.BizError) {
	cur := base.CurrentUser(ctx)
	if req.ID <= 0 {
		return nil, base.ErrBadReq("id 不能为空")
	}
	if req.Budget < 0 {
		return nil, base.ErrBadReq("额度不能为负数")
	}
	k, err := store.C().ApiKey().GetByID(req.ID)
	if err != nil {
		slog.Error("get api key failed", "err", err, "id", req.ID)
		return nil, base.ErrInternal
	}
	if k == nil || k.UserID != cur.ID {
		return nil, base.ErrNotFound("API Key 不存在")
	}
	// 校验有限额 key 的总额不超过用户余额（排除当前 key 原值）
	if berr := checkBudgetWithinUserLimit(cur, k.ID, req.Budget, req.Unlimited); berr != nil {
		return nil, berr
	}
	if err := store.C().ApiKey().UpdateBudget(k.ID, req.Budget, req.Unlimited); err != nil {
		slog.Error("update api key budget failed", "err", err, "id", k.ID)
		return nil, base.ErrInternal
	}
	k.Budget = req.Budget
	k.Unlimited = req.Unlimited
	item := toApiKeyItem(*k, "")
	return &item, nil
}

// ===== 超管：管理指定用户的 API Key =====

// ListApiKeyAdmin 超管查询指定用户的 API Key 列表（key 脱敏）
func ListApiKeyAdmin(ctx context.Context, req *ApiKeyListReq) (*base.PageResult[apiKeyItem], *base.BizError) {
	if req.UserID <= 0 {
		return nil, base.ErrBadReq("user_id 不能为空")
	}
	pc := &store.PageContext{Page: req.Page, PageSize: req.PageSize}
	keys, err := store.C().SetPage(pc).ApiKey().ListByUser(req.UserID)
	if err != nil {
		slog.Error("list api keys failed", "err", err, "user_id", req.UserID)
		return nil, base.ErrInternal
	}
	items := make([]apiKeyItem, 0, len(keys))
	for _, k := range keys {
		items = append(items, toApiKeyItem(k, ""))
	}
	return &base.PageResult[apiKeyItem]{Items: items, Total: pc.Total, Page: pc.Page, PageSize: pc.PageSize}, nil
}

// ToggleApiKeyAdmin 超管启用/禁用指定用户的 API Key
func ToggleApiKeyAdmin(ctx context.Context, req *ApiKeyIdReq) (*apiKeyItem, *base.BizError) {
	if req.ID <= 0 {
		return nil, base.ErrBadReq("id 不能为空")
	}
	k, err := store.C().ApiKey().GetByID(req.ID)
	if err != nil {
		slog.Error("get api key failed", "err", err, "id", req.ID)
		return nil, base.ErrInternal
	}
	if k == nil {
		return nil, base.ErrNotFound("API Key 不存在")
	}
	if err := store.C().ApiKey().SetEnabled(k.ID, !k.Enabled); err != nil {
		slog.Error("toggle api key failed", "err", err, "id", k.ID)
		return nil, base.ErrInternal
	}
	k.Enabled = !k.Enabled
	item := toApiKeyItem(*k, "")
	return &item, nil
}

// DeleteApiKeyAdmin 超管删除指定用户的 API Key
func DeleteApiKeyAdmin(ctx context.Context, req *ApiKeyIdReq) (*struct{}, *base.BizError) {
	if req.ID <= 0 {
		return nil, base.ErrBadReq("id 不能为空")
	}
	k, err := store.C().ApiKey().GetByID(req.ID)
	if err != nil {
		slog.Error("get api key failed", "err", err, "id", req.ID)
		return nil, base.ErrInternal
	}
	if k == nil {
		return nil, base.ErrNotFound("API Key 不存在")
	}
	if err := service.NewApiKeyService().Delete(k.ID); err != nil {
		slog.Error("delete api key failed", "err", err, "id", k.ID)
		return nil, base.ErrInternal
	}
	return &struct{}{}, nil
}

// RenameApiKeyAdmin 超管重命名指定用户的 API Key
func RenameApiKeyAdmin(ctx context.Context, req *RenameApiKeyReq) (*apiKeyItem, *base.BizError) {
	if req.ID <= 0 {
		return nil, base.ErrBadReq("id 不能为空")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, base.ErrBadReq("名称不能为空")
	}
	if len(name) > 64 {
		return nil, base.ErrBadReq("名称过长")
	}
	k, err := store.C().ApiKey().GetByID(req.ID)
	if err != nil {
		slog.Error("get api key failed", "err", err, "id", req.ID)
		return nil, base.ErrInternal
	}
	if k == nil {
		return nil, base.ErrNotFound("API Key 不存在")
	}
	if err := store.C().ApiKey().UpdateName(k.ID, name); err != nil {
		slog.Error("rename api key failed", "err", err, "id", k.ID)
		return nil, base.ErrInternal
	}
	k.Name = name
	item := toApiKeyItem(*k, "")
	return &item, nil
}

// UpdateBudgetApiKeyAdmin 超管修改指定用户的 API Key 额度。
// 按 key 归属的用户余额校验总额。
func UpdateBudgetApiKeyAdmin(ctx context.Context, req *UpdateBudgetApiKeyReq) (*apiKeyItem, *base.BizError) {
	if req.ID <= 0 {
		return nil, base.ErrBadReq("id 不能为空")
	}
	if req.Budget < 0 {
		return nil, base.ErrBadReq("额度不能为负数")
	}
	k, err := store.C().ApiKey().GetByID(req.ID)
	if err != nil {
		slog.Error("get api key failed", "err", err, "id", req.ID)
		return nil, base.ErrInternal
	}
	if k == nil {
		return nil, base.ErrNotFound("API Key 不存在")
	}
	// 用 key 归属用户做额度校验
	owner, err := store.C().User().GetByIDAny(k.UserID)
	if err != nil || owner == nil {
		slog.Error("get key owner failed", "err", err, "user_id", k.UserID)
		return nil, base.ErrInternal
	}
	if berr := checkBudgetWithinUserLimit(owner, k.ID, req.Budget, req.Unlimited); berr != nil {
		return nil, berr
	}
	if err := store.C().ApiKey().UpdateBudget(k.ID, req.Budget, req.Unlimited); err != nil {
		slog.Error("update api key budget failed", "err", err, "id", k.ID)
		return nil, base.ErrInternal
	}
	k.Budget = req.Budget
	k.Unlimited = req.Unlimited
	item := toApiKeyItem(*k, "")
	return &item, nil
}

// ===== API Key 模型访问策略 =====

// ApiKeyModelAccessReq 查询 Key 模型策略的请求
type ApiKeyModelAccessReq struct {
	ApiKeyID int64 `json:"api_key_id"`
}

// SetApiKeyModelAccessReq 设置 Key 模型策略的请求
type SetApiKeyModelAccessReq struct {
	ApiKeyID    int64   `json:"api_key_id"`
	ModelPolicy string  `json:"model_policy"` // all | whitelist
	ModelIDs    []int64 `json:"model_ids"`    // whitelist 策略下生效
}

// ApiKeyModelBrief 模型简要信息（只暴露展示字段，不含价格）
type ApiKeyModelBrief struct {
	ID       int64  `json:"id"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

// ApiKeyModelAccessResp Key 模型策略响应
type ApiKeyModelAccessResp struct {
	ModelPolicy string             `json:"model_policy"`
	ModelIDs    []int64            `json:"model_ids"`
	Models      []ApiKeyModelBrief `json:"models"` // 白名单模型详情，供前端常驻展示已选模型
}

// toModelBriefs 模型列表转简要信息，同时收集 ID 列表
func toModelBriefs(models []model.Model) ([]ApiKeyModelBrief, []int64) {
	briefs := make([]ApiKeyModelBrief, 0, len(models))
	ids := make([]int64, 0, len(models))
	for _, m := range models {
		briefs = append(briefs, ApiKeyModelBrief{ID: m.ID, Provider: m.Provider, Model: m.Model})
		ids = append(ids, m.ID)
	}
	return briefs, ids
}

// validateModelPolicy 校验策略值
func validateModelPolicy(p string) *base.BizError {
	if p != store.ModelPolicyAll && p != store.ModelPolicyWhitelist {
		return base.ErrBadReq("model_policy 只能是 all 或 whitelist")
	}
	return nil
}

// GetApiKeyModelAccessSelf 普通用户查询自己 Key 的模型策略
func GetApiKeyModelAccessSelf(ctx context.Context, req *ApiKeyModelAccessReq) (*ApiKeyModelAccessResp, *base.BizError) {
	cur := base.CurrentUser(ctx)
	if req.ApiKeyID <= 0 {
		return nil, base.ErrBadReq("api_key_id 不能为空")
	}
	k, err := store.C().ApiKey().GetByID(req.ApiKeyID)
	if err != nil {
		slog.Error("get api key failed", "err", err, "id", req.ApiKeyID)
		return nil, base.ErrInternal
	}
	if k == nil || k.UserID != cur.ID {
		return nil, base.ErrNotFound("API Key 不存在")
	}
	policy, models, err := service.NewApiKeyService().GetModelAccessDetail(req.ApiKeyID)
	if err != nil {
		slog.Error("get api key model access failed", "err", err, "id", req.ApiKeyID)
		return nil, base.ErrInternal
	}
	briefs, modelIDs := toModelBriefs(models)
	return &ApiKeyModelAccessResp{ModelPolicy: policy, ModelIDs: modelIDs, Models: briefs}, nil
}

// SetApiKeyModelAccessSelf 普通用户设置自己 Key 的模型策略
func SetApiKeyModelAccessSelf(ctx context.Context, req *SetApiKeyModelAccessReq) (*ApiKeyModelAccessResp, *base.BizError) {
	cur := base.CurrentUser(ctx)
	if req.ApiKeyID <= 0 {
		return nil, base.ErrBadReq("api_key_id 不能为空")
	}
	if berr := validateModelPolicy(req.ModelPolicy); berr != nil {
		return nil, berr
	}
	k, err := store.C().ApiKey().GetByID(req.ApiKeyID)
	if err != nil {
		slog.Error("get api key failed", "err", err, "id", req.ApiKeyID)
		return nil, base.ErrInternal
	}
	if k == nil || k.UserID != cur.ID {
		return nil, base.ErrNotFound("API Key 不存在")
	}
	modelIDs := req.ModelIDs
	if modelIDs == nil {
		modelIDs = []int64{}
	}
	if err := service.NewApiKeyService().SetModelAccess(req.ApiKeyID, req.ModelPolicy, modelIDs); err != nil {
		slog.Error("set api key model access failed", "err", err, "id", req.ApiKeyID)
		return nil, base.ErrInternal
	}
	return &ApiKeyModelAccessResp{ModelPolicy: req.ModelPolicy, ModelIDs: modelIDs}, nil
}

// GetApiKeyModelAccessAdmin 超管查询指定 Key 的模型策略
func GetApiKeyModelAccessAdmin(ctx context.Context, req *ApiKeyModelAccessReq) (*ApiKeyModelAccessResp, *base.BizError) {
	if req.ApiKeyID <= 0 {
		return nil, base.ErrBadReq("api_key_id 不能为空")
	}
	k, err := store.C().ApiKey().GetByID(req.ApiKeyID)
	if err != nil {
		slog.Error("get api key failed", "err", err, "id", req.ApiKeyID)
		return nil, base.ErrInternal
	}
	if k == nil {
		return nil, base.ErrNotFound("API Key 不存在")
	}
	policy, models, err := service.NewApiKeyService().GetModelAccessDetail(req.ApiKeyID)
	if err != nil {
		slog.Error("get api key model access failed", "err", err, "id", req.ApiKeyID)
		return nil, base.ErrInternal
	}
	briefs, modelIDs := toModelBriefs(models)
	return &ApiKeyModelAccessResp{ModelPolicy: policy, ModelIDs: modelIDs, Models: briefs}, nil
}

// SetApiKeyModelAccessAdmin 超管设置指定 Key 的模型策略
func SetApiKeyModelAccessAdmin(ctx context.Context, req *SetApiKeyModelAccessReq) (*ApiKeyModelAccessResp, *base.BizError) {
	if req.ApiKeyID <= 0 {
		return nil, base.ErrBadReq("api_key_id 不能为空")
	}
	if berr := validateModelPolicy(req.ModelPolicy); berr != nil {
		return nil, berr
	}
	k, err := store.C().ApiKey().GetByID(req.ApiKeyID)
	if err != nil {
		slog.Error("get api key failed", "err", err, "id", req.ApiKeyID)
		return nil, base.ErrInternal
	}
	if k == nil {
		return nil, base.ErrNotFound("API Key 不存在")
	}
	modelIDs := req.ModelIDs
	if modelIDs == nil {
		modelIDs = []int64{}
	}
	if err := service.NewApiKeyService().SetModelAccess(req.ApiKeyID, req.ModelPolicy, modelIDs); err != nil {
		slog.Error("set api key model access failed", "err", err, "id", req.ApiKeyID)
		return nil, base.ErrInternal
	}
	return &ApiKeyModelAccessResp{ModelPolicy: req.ModelPolicy, ModelIDs: modelIDs}, nil
}
