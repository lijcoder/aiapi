package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"

	"github.com/lijcoder/aiapi/manager/base"
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

// maskKey 对外展示的脱敏 key：保留前 6 与后 4 位，中间用 **** 替换。
func maskKey(k string) string {
	if len(k) <= 10 {
		return k
	}
	return k[:6] + "****" + k[len(k)-4:]
}

// toApiKeyItem 将模型转为视图对象。mask 控制是否对 key 脱敏。
func toApiKeyItem(k model.ApiKey, mask bool) apiKeyItem {
	key := k.Key
	if mask {
		key = maskKey(k.Key)
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
func createApiKeyRetry(userID int64, name string, budget float64, unlimited bool) (*model.ApiKey, error) {
	for i := 0; i < 3; i++ {
		plain, err := newApiKey()
		if err != nil {
			return nil, err
		}
		k := &model.ApiKey{
			UserID:    userID,
			Key:       plain,
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
			return nil, err
		}
		return k, nil
	}
	return nil, errors.New("api key generation failed after retries")
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
		return base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	// 排除当前正在修改的 key（如果它原本是有限额的，需从总和中减掉原值）
	if excludeID > 0 {
		exist, gerr := store.C().ApiKey().GetByID(excludeID)
		if gerr != nil {
			slog.Error("get api key for budget check failed", "err", gerr, "id", excludeID)
			return base.NewBizError(base.CodeUnknown, base.InternalServerError)
		}
		if exist != nil && exist.UserID == cur.ID && !exist.Unlimited {
			sum -= exist.Budget
		}
	}
	if !newUnlimited {
		sum += newBudget
	}
	if sum > cur.Budget {
		return base.NewBizError(base.CodeBudgetExceeded, "有限额密钥的额度总和不能超过账户余额")
	}
	return nil
}

// ListApiKeySelf 查询当前用户的 API Key 列表（key 脱敏）
func ListApiKeySelf(ctx context.Context) ([]apiKeyItem, *base.BizError) {
	cur := base.CurrentUser(ctx)
	keys, err := store.C().ApiKey().ListByUser(cur.ID)
	if err != nil {
		slog.Error("list api keys failed", "err", err, "user_id", cur.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	items := make([]apiKeyItem, 0, len(keys))
	for _, k := range keys {
		items = append(items, toApiKeyItem(k, true))
	}
	return items, nil
}

// CreateApiKeySelf 创建 API Key，明文 key 仅在本响应中返回一次
func CreateApiKeySelf(ctx context.Context, req *CreateApiKeyReq) (*CreateApiKeyResp, *base.BizError) {
	cur := base.CurrentUser(ctx)
	// 名称允许为空，但过长会撑爆前端；这里只做长度上限。
	if name := strings.TrimSpace(req.Name); len(name) > 64 {
		return nil, base.NewBizError(base.CodeInvalidParams, "name too long")
	}
	if req.Budget < 0 {
		return nil, base.NewBizError(base.CodeInvalidParams, "budget must be >= 0")
	}
	// 校验有限额 key 的总额不超过用户余额
	if berr := checkBudgetWithinUserLimit(cur, 0, req.Budget, req.Unlimited); berr != nil {
		return nil, berr
	}

	k, err := createApiKeyRetry(cur.ID, strings.TrimSpace(req.Name), req.Budget, req.Unlimited)
	if err != nil {
		slog.Error("create api key failed", "err", err, "user_id", cur.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	return &CreateApiKeyResp{ApiKey: toApiKeyItem(*k, false)}, nil
}

// ToggleApiKeySelf 启用/禁用 API Key。先校验归属当前用户，防止越权。
func ToggleApiKeySelf(ctx context.Context, req *ApiKeyIdReq) (*apiKeyItem, *base.BizError) {
	cur := base.CurrentUser(ctx)
	if req.ID <= 0 {
		return nil, base.NewBizError(base.CodeInvalidParams, "id is required")
	}
	k, err := store.C().ApiKey().GetByID(req.ID)
	if err != nil {
		slog.Error("get api key failed", "err", err, "id", req.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	if k == nil || k.UserID != cur.ID {
		return nil, base.NewBizError(base.CodeApiKeyNotFound, "api key not found")
	}
	if err := store.C().ApiKey().SetEnabled(k.ID, !k.Enabled); err != nil {
		slog.Error("toggle api key failed", "err", err, "id", k.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	k.Enabled = !k.Enabled
	item := toApiKeyItem(*k, true)
	return &item, nil
}

// DeleteApiKeySelf 删除 API Key。先校验归属当前用户，防止越权。
func DeleteApiKeySelf(ctx context.Context, req *ApiKeyIdReq) (*struct{}, *base.BizError) {
	cur := base.CurrentUser(ctx)
	if req.ID <= 0 {
		return nil, base.NewBizError(base.CodeInvalidParams, "id is required")
	}
	k, err := store.C().ApiKey().GetByID(req.ID)
	if err != nil {
		slog.Error("get api key failed", "err", err, "id", req.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	if k == nil || k.UserID != cur.ID {
		return nil, base.NewBizError(base.CodeApiKeyNotFound, "api key not found")
	}
	if err := store.C().ApiKey().Delete(k.ID); err != nil {
		slog.Error("delete api key failed", "err", err, "id", k.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	return &struct{}{}, nil
}

// RenameApiKeySelf 修改 API Key 名称。先校验归属当前用户，防止越权。
func RenameApiKeySelf(ctx context.Context, req *RenameApiKeyReq) (*apiKeyItem, *base.BizError) {
	cur := base.CurrentUser(ctx)
	if req.ID <= 0 {
		return nil, base.NewBizError(base.CodeInvalidParams, "id is required")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, base.NewBizError(base.CodeInvalidParams, "name is required")
	}
	if len(name) > 64 {
		return nil, base.NewBizError(base.CodeInvalidParams, "name too long")
	}
	k, err := store.C().ApiKey().GetByID(req.ID)
	if err != nil {
		slog.Error("get api key failed", "err", err, "id", req.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	if k == nil || k.UserID != cur.ID {
		return nil, base.NewBizError(base.CodeApiKeyNotFound, "api key not found")
	}
	if err := store.C().ApiKey().UpdateName(k.ID, name); err != nil {
		slog.Error("rename api key failed", "err", err, "id", k.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	k.Name = name
	item := toApiKeyItem(*k, true)
	return &item, nil
}

// UpdateBudgetApiKeySelf 修改 API Key 的额度与限额模式。
// unlimited=false 时 budget 生效，且所有“有限额” key 的 budget 之和不能超过用户余额。
// 先校验归属当前用户，防止越权。
func UpdateBudgetApiKeySelf(ctx context.Context, req *UpdateBudgetApiKeyReq) (*apiKeyItem, *base.BizError) {
	cur := base.CurrentUser(ctx)
	if req.ID <= 0 {
		return nil, base.NewBizError(base.CodeInvalidParams, "id is required")
	}
	if req.Budget < 0 {
		return nil, base.NewBizError(base.CodeInvalidParams, "budget must be >= 0")
	}
	k, err := store.C().ApiKey().GetByID(req.ID)
	if err != nil {
		slog.Error("get api key failed", "err", err, "id", req.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	if k == nil || k.UserID != cur.ID {
		return nil, base.NewBizError(base.CodeApiKeyNotFound, "api key not found")
	}
	// 校验有限额 key 的总额不超过用户余额（排除当前 key 原值）
	if berr := checkBudgetWithinUserLimit(cur, k.ID, req.Budget, req.Unlimited); berr != nil {
		return nil, berr
	}
	if err := store.C().ApiKey().UpdateBudget(k.ID, req.Budget, req.Unlimited); err != nil {
		slog.Error("update api key budget failed", "err", err, "id", k.ID)
		return nil, base.NewBizError(base.CodeUnknown, base.InternalServerError)
	}
	k.Budget = req.Budget
	k.Unlimited = req.Unlimited
	item := toApiKeyItem(*k, true)
	return &item, nil
}
