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
	ID        int64     `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Account   string    `db:"account" json:"account"`
	Password  string    `db:"password" json:"-"` // bcrypt 哈希，不对外输出
	Budget    float64   `db:"budget" json:"budget"`
	Unlimited bool      `db:"unlimited" json:"unlimited"`
	Enabled   bool      `db:"enabled" json:"enabled"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
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

// UserSession 用户登录会话
type UserSession struct {
	ID        int64     `db:"id" json:"id"`
	Token     string    `db:"token" json:"-"`
	UserID    int64     `db:"user_id" json:"user_id"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// ApiKey API 密钥
type ApiKey struct {
	ID        int64     `db:"id" json:"id"`
	UserID    int64     `db:"user_id" json:"user_id"`
	Key       string    `db:"key" json:"key"`
	Name      string    `db:"name" json:"name"`
	Budget    float64   `db:"budget" json:"budget"`
	Unlimited bool      `db:"unlimited" json:"unlimited"`
	Enabled   bool      `db:"enabled" json:"enabled"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
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
	CreatedAt           time.Time `db:"created_at" json:"created_at"`
}

// UsageRecord token 使用记录
type UsageRecord struct {
	ID              int64     `db:"id" json:"id"`
	UserID          int64     `db:"user_id" json:"user_id"`
	ApiKey          string    `db:"api_key" json:"api_key"`
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
	ApiKey         string    `db:"api_key" json:"api_key"`
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
