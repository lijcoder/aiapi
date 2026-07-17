package store

import "time"

// Provider 大模型提供商配置
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

// User 用户
type User struct {
	ID        int64     `db:"id" json:"id"`
	Account   string    `db:"account" json:"account"`
	Name      string    `db:"name" json:"name"`
	Budget    float64   `db:"budget" json:"budget"`
	Unlimited bool      `db:"unlimited" json:"unlimited"`
	Enabled   bool      `db:"enabled" json:"enabled"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// ApiKey 用户 API Key
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

// UsageRecord Token 用量记录
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

// RequestLog HTTP 请求日志
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
