package service

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/lijcoder/aiapi/store"
	"github.com/lijcoder/aiapi/store/model"
)

const testPricingConfig = `{
  "version": 1,
  "default_price": {"input_cache_hit": 1, "input_cache_miss": 2, "output": 3},
  "rules": [
    {
      "id": "weekday-large",
      "name": "工作日高峰大请求",
      "enabled": true,
      "priority": 100,
      "when": {
        "time": {"weekdays": [1, 2, 3, 4, 5], "windows": [{"start": "09:00", "end": "12:00"}]},
        "total_tokens": {"gte": 10000, "lt": 50000}
      },
      "price": {"input_cache_hit": 4, "input_cache_miss": 5, "output": 6}
    },
    {
      "id": "large",
      "name": "大请求",
      "enabled": true,
      "priority": 50,
      "when": {"total_tokens": {"gt": 10000}},
      "price": {"input_cache_hit": 7, "input_cache_miss": 8, "output": 9}
    }
  ]
}`

func TestQuoteModelPricing_CombinedConditionsAndPriority(t *testing.T) {
	m := &model.Model{ID: 12, PricingConfig: testPricingConfig}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	weekdayMorning := time.Date(2026, time.September, 14, 10, 0, 0, 0, loc) // Monday

	quote, err := QuoteModelPricing(m, weekdayMorning, 20000)
	if err != nil {
		t.Fatalf("quote: %v", err)
	}
	if quote.Snapshot.Source != "rule" || quote.Snapshot.Rule == nil || quote.Snapshot.Rule.ID != "weekday-large" {
		t.Fatalf("unexpected matched rule: %+v", quote.Snapshot)
	}
	if quote.Price.Output != 6 {
		t.Fatalf("output price = %v, want 6", quote.Price.Output)
	}

	quote, err = QuoteModelPricing(m, weekdayMorning, 50000)
	if err != nil {
		t.Fatalf("quote upper boundary: %v", err)
	}
	if quote.Snapshot.Rule == nil || quote.Snapshot.Rule.ID != "large" {
		t.Fatalf("upper boundary should match large rule, got %+v", quote.Snapshot.Rule)
	}

	quote, err = QuoteModelPricing(m, weekdayMorning, 10000)
	if err != nil {
		t.Fatalf("quote lower boundary: %v", err)
	}
	if quote.Snapshot.Rule == nil || quote.Snapshot.Rule.ID != "weekday-large" {
		t.Fatalf("lower boundary should include weekday-large, got %+v", quote.Snapshot.Rule)
	}

	weekend := time.Date(2026, time.September, 13, 10, 0, 0, 0, loc) // Sunday
	quote, err = QuoteModelPricing(m, weekend, 9000)
	if err != nil {
		t.Fatalf("quote default: %v", err)
	}
	if quote.Snapshot.Source != "default" || quote.Snapshot.Rule != nil || quote.Price.Output != 3 {
		t.Fatalf("want default quote, got %+v", quote)
	}
}

func TestPricingConfigValidation(t *testing.T) {
	for _, tc := range []struct {
		name string
		json string
	}{
		{name: "unknown field", json: `{"version":1,"default_price":{"input_cache_hit":0,"input_cache_miss":0,"output":0},"rules":[],"extra":true}`},
		{name: "duplicate priority", json: `{"version":1,"default_price":{"input_cache_hit":0,"input_cache_miss":0,"output":0},"rules":[{"id":"a","enabled":true,"priority":1,"when":{"total_tokens":{"gt":1}},"price":{"input_cache_hit":0,"input_cache_miss":0,"output":0}},{"id":"b","enabled":true,"priority":1,"when":{"total_tokens":{"lt":2}},"price":{"input_cache_hit":0,"input_cache_miss":0,"output":0}}]}`},
		{name: "empty token interval", json: `{"version":1,"default_price":{"input_cache_hit":0,"input_cache_miss":0,"output":0},"rules":[{"id":"a","enabled":true,"priority":1,"when":{"total_tokens":{"gt":2,"lte":2}},"price":{"input_cache_hit":0,"input_cache_miss":0,"output":0}}]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParsePricingConfig(tc.json); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}

	normalized, err := NormalizePricingConfig(`{"version":1,"timezone":"","default_price":{"input_cache_hit":0,"input_cache_miss":0,"output":0},"rules":[]}`)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if !strings.Contains(normalized, `"timezone":"Asia/Shanghai"`) {
		t.Fatalf("timezone default missing: %s", normalized)
	}
}

func TestPricingV2ConditionTree(t *testing.T) {
	raw := `{"version":2,"timezone":"Asia/Shanghai","default_price":{"input_cache_hit":1,"input_cache_miss":2,"output":3},"rules":[{"name":"工作日或节日大请求","enabled":true,"priority":10,"when":{"op":"and","children":[{"type":"weekday","values":[1,2,3,4,5]},{"op":"or","children":[{"type":"time_range","start":"09:00","end":"12:00"},{"type":"month_day","month_days":["12-25"]}]},{"op":"or","children":[{"type":"total_tokens","operator":"gt","value":100},{"type":"total_tokens","operator":"lt","value":20}]}]},"price":{"input_cache_hit":4,"input_cache_miss":5,"output":6}}]}`
	m := &model.Model{ID: 1, PricingConfig: raw}
	loc := mustShanghai(t)
	for _, tc := range []struct {
		name string
		at   time.Time
		tok  int
		want bool
	}{
		{"weekday time gt", time.Date(2026, time.September, 14, 10, 0, 0, 0, loc), 101, true},
		{"weekday time lt", time.Date(2026, time.September, 14, 10, 0, 0, 0, loc), 10, true},
		{"weekday outside token", time.Date(2026, time.September, 14, 10, 0, 0, 0, loc), 50, false},
		{"month day", time.Date(2026, time.December, 25, 20, 0, 0, 0, loc), 10, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			quote, err := QuoteModelPricing(m, tc.at, tc.tok)
			if err != nil {
				t.Fatal(err)
			}
			if (quote.Snapshot.Source == "rule") != tc.want {
				t.Fatalf("matched=%v want %v snapshot=%+v", quote.Snapshot.Source == "rule", tc.want, quote.Snapshot)
			}
		})
	}
	if got, err := NormalizePricingConfig(raw); err != nil || strings.Contains(got, `"id"`) || !strings.Contains(got, `"version":2`) {
		t.Fatalf("normalized=%s err=%v", got, err)
	}
}

func setupBillingTestDB(t *testing.T) {
	t.Helper()
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.MustExec(`CREATE TABLE users (id INTEGER PRIMARY KEY, budget REAL NOT NULL)`)
	db.MustExec(`CREATE TABLE api_keys (id INTEGER PRIMARY KEY, budget REAL NOT NULL)`)
	db.MustExec(`CREATE TABLE usage_records (
		id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, api_key_id INTEGER NOT NULL,
		provider TEXT NOT NULL, model TEXT NOT NULL, input_tokens INTEGER NOT NULL, output_tokens INTEGER NOT NULL,
		total_tokens INTEGER NOT NULL, request_id TEXT, stream INTEGER, cached_tokens INTEGER, reasoning_tokens INTEGER,
		cost REAL, unlimited INTEGER, pricing_snapshot TEXT NOT NULL, first_token_ms INTEGER, latency_ms INTEGER,
		created_at DATETIME
	)`)
	db.MustExec(`INSERT INTO users (id, budget) VALUES (1, 10)`)
	db.MustExec(`INSERT INTO api_keys (id, budget) VALUES (2, 8)`)
	if err := store.Init(db); err != nil {
		t.Fatalf("store init: %v", err)
	}
}

func TestBillingServiceRecordUsage_WritesSnapshotAndDeductsAtomically(t *testing.T) {
	setupBillingTestDB(t)
	m := &model.Model{ID: 12, PricingConfig: testPricingConfig}
	record, err := NewBillingService().RecordUsage(UsageBillingInput{
		Model: m, UserID: 1, APIKeyID: 2, Provider: "openai", ModelName: "gpt-test",
		InputTokens: 1000000, CachedTokens: 200000, OutputTokens: 1000000, TotalTokens: 20000,
		RequestStartedAt: time.Date(2026, time.September, 14, 10, 0, 0, 0, mustShanghai(t)),
	})
	if err != nil {
		t.Fatalf("record usage: %v", err)
	}
	// 0.2*4 + 0.8*5 + 1*6 = 10.8
	if record.Cost != 10.8 {
		t.Fatalf("cost = %v, want 10.8", record.Cost)
	}
	var snapshot PricingSnapshot
	if err := json.Unmarshal([]byte(record.PricingSnapshot), &snapshot); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if snapshot.Rule == nil || snapshot.Rule.ID != "weekday-large" || snapshot.Price.Output != 6 {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
	if snapshot.MatchedRuleName != "工作日高峰大请求" {
		t.Fatalf("matched rule name = %q", snapshot.MatchedRuleName)
	}
	if snapshot.RequestStartedAt != "2026-09-14 10:00:00" {
		t.Fatalf("request started at = %q", snapshot.RequestStartedAt)
	}
	if snapshot.PricingConfig.DefaultPrice.Output != 3 || len(snapshot.PricingConfig.Rules) != 2 {
		t.Fatalf("snapshot should contain full pricing config: %+v", snapshot.PricingConfig)
	}

	var count int
	if err := store.C().Query(`SELECT COUNT(*) FROM usage_records`, nil).Get(&count); err != nil || count != 1 {
		t.Fatalf("usage record count = %d, err=%v", count, err)
	}
	var userBudget, keyBudget float64
	if err := store.C().Query(`SELECT budget FROM users WHERE id=1`, nil).Get(&userBudget); err != nil {
		t.Fatal(err)
	}
	if err := store.C().Query(`SELECT budget FROM api_keys WHERE id=2`, nil).Get(&keyBudget); err != nil {
		t.Fatal(err)
	}
	if math.Abs(userBudget+0.8) > 1e-9 || math.Abs(keyBudget+2.8) > 1e-9 {
		t.Fatalf("budgets = %v/%v, want -0.8/-2.8", userBudget, keyBudget)
	}
}

func TestBillingServiceRecordUsage_RejectsInvalidUsage(t *testing.T) {
	_, err := NewBillingService().RecordUsage(UsageBillingInput{
		Model:        &model.Model{PricingConfig: testPricingConfig},
		InputTokens:  10,
		CachedTokens: 11,
	})
	if err == nil || err.Error() != "用量数据无效" {
		t.Fatalf("err = %v, want invalid usage error", err)
	}
}

func mustShanghai(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}
