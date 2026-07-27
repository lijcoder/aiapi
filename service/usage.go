package service

import "github.com/lijcoder/aiapi/store"

// UsageService 封装用量统计业务逻辑。
type UsageService struct{}

// NewUsageService 创建 UsageService。
func NewUsageService() *UsageService { return &UsageService{} }

// StatsByUser 统计汇总：按时间或维度分组，排序方向由 groupBy 决定（时间 ASC，维度 DESC）。
func (s *UsageService) StatsByUser(userID int64, mode, startDate, endDate, apiKey, mdl, provider, groupBy string) ([]store.UsageStatRow, error) {
	orderDir := "ASC"
	if groupBy != "" {
		orderDir = "DESC"
	}
	return store.C().Usage().StatsByUser(userID, mode, startDate, endDate, apiKey, mdl, provider, groupBy, orderDir)
}

// StatsByAdmin 全局统计汇总，排序方向由 groupBy 决定（时间 ASC，维度 DESC）。
func (s *UsageService) StatsByAdmin(userID int64, mode, startDate, endDate, apiKey, mdl, provider, groupBy string) ([]store.UsageStatRow, error) {
	orderDir := "ASC"
	if groupBy != "" {
		orderDir = "DESC"
	}
	return store.C().Usage().StatsByAdmin(userID, mode, startDate, endDate, apiKey, mdl, provider, groupBy, orderDir)
}
