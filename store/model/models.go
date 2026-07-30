package model

import (
	"encoding/json"
	"time"
)

// Provider 上游服务配置
type Provider struct {
	ID        int64     `db:"id" json:"id"`
	Type      string    `db:"type" json:"type"`
	Config    string    `db:"config" json:"config"`
	Enabled   bool      `db:"enabled" json:"enabled"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// ProviderConfig provider.config 字段的解析结构
type ProviderConfig struct {
	Domain  string              `json:"domain"`
	Headers map[string][]string `json:"headers"`
}

// ParseConfig 解析 provider.config JSON
func (p *Provider) ParseConfig() (*ProviderConfig, error) {
	var cfg ProviderConfig
	if err := json.Unmarshal([]byte(p.Config), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// User 用户
type User struct {
	ID         int64     `db:"id" json:"id"`
	Name       string    `db:"name" json:"name"`
	Account    string    `db:"account" json:"account"`
	Password   string    `db:"password" json:"-"`    // bcrypt 哈希，不对外输出
	TotpSecret string    `db:"totp_secret" json:"-"` // TOTP 密钥（AES-GCM 加密存储，空=未开启 2FA），不对外输出
	Email      string    `db:"email" json:"email"`
	Budget     float64   `db:"budget" json:"budget"`
	Unlimited  bool      `db:"unlimited" json:"unlimited"`
	Enabled    bool      `db:"enabled" json:"enabled"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

// Role 角色
type Role struct {
	ID        int64     `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Code      string    `db:"code" json:"code,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// UserRole 用户-角色关联
type UserRole struct {
	ID        int64     `db:"id" json:"id"`
	UserID    int64     `db:"user_id" json:"user_id"`
	RoleID    int64     `db:"role_id" json:"role_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// RolePermission 接口级权限 (subject=role_id, entity, action, value)
type RolePermission struct {
	ID        int64     `db:"id" json:"id"`
	RoleID    int64     `db:"role_id" json:"role_id"`
	Entity    string    `db:"entity" json:"entity"`
	Action    string    `db:"action" json:"action"`
	Value     string    `db:"value" json:"value"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// UserSession 用户登录会话（refresh token 存储）
type UserSession struct {
	ID                int64     `db:"id" json:"id"`
	Token             string    `db:"token" json:"-"`     // refresh token 的 SHA-256 哈希
	FamilyID          string    `db:"family_id" json:"-"` // 登录链标识，重用检测用
	UserID            int64     `db:"user_id" json:"user_id"`
	ExpiresAt         time.Time `db:"expires_at" json:"expires_at"`                   // 滑动过期时间
	AbsoluteExpiresAt time.Time `db:"absolute_expires_at" json:"absolute_expires_at"` // 绝对过期上限
	UA                string    `db:"ua" json:"ua"`                                   // User-Agent 摘要
	IP                string    `db:"ip" json:"ip"`                                   // 登录 IP
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
}

// ApiKey API 密钥
// key 原文不直接明文落库：KeyHash 用于鉴权比对，KeyShow 用于展示识别（如 sk-abcdef1）。
// KeyEnc 为 key 原文的 AES-256-GCM 密文，可还原（查看接口解密返回）；
// 空串表示旧版本创建的 key（仅存 hash，无法还原）。
// 明文 key 仅在创建时返回给用户一次，此后可通过查看接口还原。
type ApiKey struct {
	ID          int64     `db:"id" json:"id"`
	UserID      int64     `db:"user_id" json:"user_id"`
	KeyHash     string    `db:"key_hash" json:"-"` // SHA-256(hex)，不对外输出
	KeyEnc      string    `db:"key_enc" json:"-"`  // AES-256-GCM 密文(base64)，空串=旧版本不可还原
	KeyShow     string    `db:"key_show" json:"-"` // 展示串（sk-abc****xyz），由 store.ApiKeyShow 生成
	Name        string    `db:"name" json:"name"`
	Budget      float64   `db:"budget" json:"budget"`
	Unlimited   bool      `db:"unlimited" json:"unlimited"`
	Enabled     bool      `db:"enabled" json:"enabled"`
	ModelPolicy string    `db:"model_policy" json:"model_policy"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// Model 模型价格配置
type Model struct {
	ID                  int64     `db:"id" json:"id"`
	Provider            string    `db:"provider" json:"provider"`
	Model               string    `db:"model" json:"model"`
	InputCacheHitPrice  float64   `db:"input_cache_hit_price" json:"input_cache_hit_price"`
	InputCacheMissPrice float64   `db:"input_cache_miss_price" json:"input_cache_miss_price"`
	OutputPrice         float64   `db:"output_price" json:"output_price"`
	MaxContextTokens    int       `db:"max_context_tokens" json:"max_context_tokens"`
	MaxCompletionTokens int       `db:"max_completion_tokens" json:"max_completion_tokens"`
	SupportsText        bool      `db:"supports_text" json:"supports_text"`
	SupportsImage       bool      `db:"supports_image" json:"supports_image"`
	SupportsVideo       bool      `db:"supports_video" json:"supports_video"`
	CreatedAt           time.Time `db:"created_at" json:"created_at"`
}

// UsageRecord token 使用记录
type UsageRecord struct {
	ID              int64     `db:"id" json:"id"`
	UserID          int64     `db:"user_id" json:"user_id"`
	ApiKeyID        int64     `db:"api_key_id" json:"api_key_id"` // 关联 api_keys.id，不存 key 原文
	Provider        string    `db:"provider" json:"provider"`
	Model           string    `db:"model" json:"model"`
	InputTokens     int       `db:"input_tokens" json:"input_tokens"`
	OutputTokens    int       `db:"output_tokens" json:"output_tokens"`
	TotalTokens     int       `db:"total_tokens" json:"total_tokens"`
	RequestID       string    `db:"request_id" json:"request_id"`
	Stream          bool      `db:"stream" json:"stream"`
	CachedTokens    int       `db:"cached_tokens" json:"cached_tokens"`
	ReasoningTokens int       `db:"reasoning_tokens" json:"reasoning_tokens"`
	Cost            float64   `db:"cost" json:"cost"`
	Unlimited       bool      `db:"unlimited" json:"unlimited"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
}

// RequestLog 请求日志
type RequestLog struct {
	ID             int64     `db:"id" json:"id"`
	ApiKeyID       int64     `db:"api_key_id" json:"api_key_id"` // 关联 api_keys.id，不存 key 原文
	Format         string    `db:"format" json:"format"`
	Provider       string    `db:"provider" json:"provider"`
	Path           string    `db:"path" json:"path"`
	StatusCode     int       `db:"status_code" json:"status_code"`
	RequestHeaders string    `db:"request_headers" json:"request_headers"`
	RequestBody    string    `db:"request_body" json:"request_body"`
	ResponseBody   string    `db:"response_body" json:"response_body"`
	Model          string    `db:"model" json:"model"`
	InputTokens    int       `db:"input_tokens" json:"input_tokens"`
	OutputTokens   int       `db:"output_tokens" json:"output_tokens"`
	TotalTokens    int       `db:"total_tokens" json:"total_tokens"`
	Error          string    `db:"error" json:"error"`
	LatencyMs      int64     `db:"latency_ms" json:"latency_ms"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}

// RechargeRecord 充值记录
type RechargeRecord struct {
	ID            int64     `db:"id" json:"id"`
	UserID        int64     `db:"user_id" json:"user_id"`
	UserName      string    `db:"user_name" json:"user_name,omitempty"`
	Amount        float64   `db:"amount" json:"amount"`
	BalanceBefore float64   `db:"balance_before" json:"balance_before"`
	BalanceAfter  float64   `db:"balance_after" json:"balance_after"`
	Operator      string    `db:"operator" json:"operator"`
	OperatorName  string    `db:"operator_name" json:"operator_name,omitempty"`
	Remark        string    `db:"remark" json:"remark"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

// Menu 菜单
type Menu struct {
	ID        int64     `db:"id" json:"id"`
	ParentID  int64     `db:"parent_id" json:"parent_id"`
	Name      string    `db:"name" json:"name"`
	Path      string    `db:"path" json:"path"`
	Icon      string    `db:"icon" json:"icon,omitempty"`
	SortOrder int       `db:"sort_order" json:"sort_order"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// RoleMenu 角色-菜单关联
type RoleMenu struct {
	ID        int64     `db:"id" json:"id"`
	RoleID    int64     `db:"role_id" json:"role_id"`
	MenuID    int64     `db:"menu_id" json:"menu_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// ApiKeyModelAccess API Key 模型白名单明细（model_policy='whitelist' 时生效）
type ApiKeyModelAccess struct {
	ID        int64     `db:"id" json:"id"`
	ApiKeyID  int64     `db:"api_key_id" json:"api_key_id"`
	ModelID   int64     `db:"model_id" json:"model_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
