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
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	ApiKey        string    `json:"api_key"`
	Provider      string    `json:"provider"`       // openai / gemini / anthropic
	Model         string    `json:"model"`           // gpt-4, claude-3.5-sonnet 等
	InputTokens   int       `json:"input_tokens"`
	OutputTokens  int       `json:"output_tokens"`
	TotalTokens   int       `json:"total_tokens"`
	RequestID     string    `json:"request_id"`      // 上游返回的 request id
	Stream        bool      `json:"stream"`          // 是否流式
	CreatedAt     time.Time `json:"created_at"`
}
