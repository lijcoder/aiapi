package db

import "time"

// Provider 大模型提供商配置
type Provider struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"`   // openai / gemini / anthropic
	Config    string    `json:"config"` // JSON: {"domain":"https://api.openai.com","headers":{"Authorization":["Bearer xxx"]}}
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProviderConfig provider.config 字段的解析结构
type ProviderConfig struct {
	Domain  string              `json:"domain"`
	Headers map[string][]string `json:"headers"`
}

// User 用户
type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Budget    float64   `json:"budget"`     // 预算上限（USD），0 表示无限制
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

// ApiKey 用户 API Key，一个用户可以有多个 Key
type ApiKey struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Key       string    `json:"key"`
	Name      string    `json:"name"`     // Key 的备注名
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

// UsageRecord Token 用量记录
type UsageRecord struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	ApiKey          string    `json:"api_key"`
	Provider        string    `json:"provider"`
	Model           string    `json:"model"`
	InputTokens     int       `json:"input_tokens"`
	OutputTokens    int       `json:"output_tokens"`
	TotalTokens     int       `json:"total_tokens"`
	RequestID       string    `json:"request_id"`
	Stream          bool      `json:"stream"`
	CachedTokens    int       `json:"cached_tokens"`
	ReasoningTokens int       `json:"reasoning_tokens"`
	CreatedAt       time.Time `json:"created_at"`
}

// RequestLog HTTP 请求日志
type RequestLog struct {
	ID            int64     `json:"id"`
	ApiKey        string    `json:"api_key"`
	Format        string    `json:"format"`         // openai/gemini/anthropic
	Provider      string    `json:"provider"`       // 后端 provider type
	Method        string    `json:"method"`
	Path          string    `json:"path"`
	StatusCode    int       `json:"status_code"`
	RequestHeaders string   `json:"request_headers"` // JSON
	RequestBody   string    `json:"request_body"`
	ResponseBody  string    `json:"response_body"`
	Model         string    `json:"model"`
	InputTokens   int       `json:"input_tokens"`
	OutputTokens  int       `json:"output_tokens"`
	TotalTokens   int       `json:"total_tokens"`
	Error         string    `json:"error"`
	LatencyMs     int64     `json:"latency_ms"`
	CreatedAt     time.Time `json:"created_at"`
}

// ModelPricing 模型价格配置
type ModelPricing struct {
	ID                int64     `json:"id"`
	Provider          string    `json:"provider"`
	Model             string    `json:"model"`
	InputCacheHitPrice  float64 `json:"input_cache_hit_price"`
	InputCacheMissPrice float64 `json:"input_cache_miss_price"`
	OutputPrice         float64 `json:"output_price"`
	Enabled           bool      `json:"enabled"`
	CreatedAt         time.Time `json:"created_at"`
}
