package parser

import (
	"encoding/json"
	"testing"
	"time"
)

func TestResponsesParser_ParseModel(t *testing.T) {
	p := &ResponsesParser{}
	if got := p.ParseModel([]byte(`{"model":"gpt-4o","input":"hi"}`)); got != "gpt-4o" {
		t.Errorf("ParseModel = %q, want gpt-4o", got)
	}
	if got := p.ParseModel([]byte(`{invalid`)); got != "" {
		t.Errorf("ParseModel on invalid body = %q, want empty", got)
	}
}

func TestResponsesParser_ParseApiKey(t *testing.T) {
	p := &ResponsesParser{}
	headers := map[string][]string{
		"Authorization": {"Bearer sk-abc123"},
	}
	if got := p.ParseApiKey(headers); got != "sk-abc123" {
		t.Errorf("ParseApiKey = %q, want sk-abc123", got)
	}
}

func TestResponsesParser_ParseUsage(t *testing.T) {
	p := &ResponsesParser{}
	body := []byte(`{
		"id":"resp_123","object":"response","model":"gpt-4o",
		"usage":{
			"input_tokens":100,"output_tokens":50,"total_tokens":150,
			"input_tokens_details":{"cached_tokens":30},
			"output_tokens_details":{"reasoning_tokens":10}
		}
	}`)

	u, err := p.ParseUsage(body)
	if err != nil {
		t.Fatalf("ParseUsage error: %v", err)
	}
	if u == nil {
		t.Fatal("ParseUsage = nil, want usage")
	}
	if u.Provider != FormatResponses {
		t.Errorf("Provider = %q, want %q", u.Provider, FormatResponses)
	}
	if u.InputTokens != 100 || u.OutputTokens != 50 || u.TotalTokens != 150 {
		t.Errorf("tokens = %d/%d/%d, want 100/50/150", u.InputTokens, u.OutputTokens, u.TotalTokens)
	}
	if u.CachedTokens != 30 {
		t.Errorf("CachedTokens = %d, want 30", u.CachedTokens)
	}
	if u.ReasoningTokens != 10 {
		t.Errorf("ReasoningTokens = %d, want 10", u.ReasoningTokens)
	}
	if u.RequestID != "resp_123" {
		t.Errorf("RequestID = %q, want resp_123", u.RequestID)
	}
	if u.Model != "gpt-4o" {
		t.Errorf("Model = %q, want gpt-4o", u.Model)
	}
}

func TestResponsesParser_ParseUsage_NotResponseObject(t *testing.T) {
	p := &ResponsesParser{}
	// 上游错误体等非 response 对象不解析为用量
	body := []byte(`{"error":{"message":"bad request"}}`)
	u, err := p.ParseUsage(body)
	if err != nil {
		t.Fatalf("ParseUsage error: %v", err)
	}
	if u != nil {
		t.Errorf("ParseUsage on error body = %v, want nil", u)
	}
}

func TestResponsesParser_ParseUsage_TotalTokensFallback(t *testing.T) {
	p := &ResponsesParser{}
	body := []byte(`{
		"id":"resp_1","object":"response","model":"gpt-4o",
		"usage":{"input_tokens":12,"output_tokens":8,"total_tokens":0}
	}`)
	u, err := p.ParseUsage(body)
	if err != nil {
		t.Fatalf("ParseUsage error: %v", err)
	}
	if u == nil {
		t.Fatal("ParseUsage = nil, want usage")
	}
	if u.TotalTokens != 20 {
		t.Errorf("TotalTokens = %d, want fallback 20", u.TotalTokens)
	}
}

func TestResponsesParser_ParseStreamUsage(t *testing.T) {
	p := &ResponsesParser{}
	// 只看 response.completed：created 里刻意放了不同的 id / model，用于证明它不参与取值
	body := []byte("event: response.created\n" +
		`data: {"type":"response.created","response":{"id":"resp_created","model":"gpt-4o-mini"}}` + "\n\n" +
		"event: response.output_text.delta\n" +
		`data: {"type":"response.output_text.delta","delta":"Hello"}` + "\n\n" +
		"event: response.completed\n" +
		`data: {"type":"response.completed","response":{"id":"resp_2","model":"gpt-4o","usage":{"input_tokens":200,"output_tokens":80,"total_tokens":280,"input_tokens_details":{"cached_tokens":40},"output_tokens_details":{"reasoning_tokens":5}}}}` + "\n\n")

	u, err := p.ParseStreamUsage(body)
	if err != nil {
		t.Fatalf("ParseStreamUsage error: %v", err)
	}
	if u == nil {
		t.Fatal("usage is nil")
	}
	if u.Provider != FormatResponses {
		t.Errorf("Provider = %q, want %q", u.Provider, FormatResponses)
	}
	if u.InputTokens != 200 || u.OutputTokens != 80 || u.TotalTokens != 280 {
		t.Errorf("tokens = %d/%d/%d, want 200/80/280", u.InputTokens, u.OutputTokens, u.TotalTokens)
	}
	if u.CachedTokens != 40 || u.ReasoningTokens != 5 {
		t.Errorf("details = cached %d / reasoning %d, want 40/5", u.CachedTokens, u.ReasoningTokens)
	}
	// identity 取自 completed（created 里的 resp_created/gpt-4o-mini 被忽略）
	if u.RequestID != "resp_2" || u.Model != "gpt-4o" {
		t.Errorf("identity = request %q / model %q, want resp_2/gpt-4o", u.RequestID, u.Model)
	}
}

// completed 未给 total_tokens 时按 input + output 回退
func TestResponsesParser_ParseStreamUsage_TotalFallback(t *testing.T) {
	p := &ResponsesParser{}
	body := []byte("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_4\",\"model\":\"gpt-4o\",\"usage\":{\"input_tokens\":12,\"output_tokens\":8,\"total_tokens\":0}}}\n\n")

	u, err := p.ParseStreamUsage(body)
	if err != nil {
		t.Fatalf("ParseStreamUsage error: %v", err)
	}
	if u == nil {
		t.Fatal("usage is nil")
	}
	if u.TotalTokens != 20 {
		t.Errorf("TotalTokens = %d, want fallback 20", u.TotalTokens)
	}
}

// 只有 identity（created）或异常/中间事件时不记用量
func TestResponsesParser_ParseStreamUsage_NoUsage(t *testing.T) {
	p := &ResponsesParser{}
	cases := map[string]string{
		"created only": "data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_1\",\"model\":\"gpt-4o\"}}\n\n",
		"created + failed": "data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_1\",\"model\":\"gpt-4o\"}}\n\n" +
			"data: {\"type\":\"response.failed\",\"response\":{\"id\":\"resp_1\",\"status\":\"failed\"}}\n\n",
		"failed":     "data: {\"type\":\"response.failed\",\"response\":{\"id\":\"resp_3\",\"status\":\"failed\"}}\n\n",
		"incomplete": "data: {\"type\":\"response.incomplete\",\"response\":{\"id\":\"resp_3\"}}\n\n",
		"intermediate": "data: {\"type\":\"response.in_progress\",\"response\":{\"id\":\"resp_1\"}}\n\n" +
			"data: {\"type\":\"response.output_text.done\",\"text\":\"Hello\"}\n\n" +
			"data: {\"type\":\"response.output_item.added\",\"output_index\":0}\n\n" +
			"data: {\"type\":\"response.content_part.done\",\"content_index\":0}\n\n",
		"empty body": "",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			u, err := p.ParseStreamUsage([]byte(body))
			if err != nil {
				t.Fatalf("ParseStreamUsage error: %v", err)
			}
			if u != nil {
				t.Errorf("usage = %+v, want nil", u)
			}
		})
	}
}

func TestResponsesParser_FormatModels(t *testing.T) {
	p := &ResponsesParser{}
	items := []ModelItem{
		{ID: "gpt-4o", CreatedAt: time.Unix(1715000000, 0), OwnedBy: "openai"},
		{ID: "gpt-4o-mini", CreatedAt: time.Unix(1715000100, 0), OwnedBy: "openai"},
	}

	body, err := p.FormatModels(items)
	if err != nil {
		t.Fatalf("FormatModels error: %v", err)
	}

	var got struct {
		Object string `json:"object"`
		Data   []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal FormatModels result: %v", err)
	}
	if got.Object != "list" {
		t.Errorf("object = %q, want list", got.Object)
	}
	if len(got.Data) != 2 {
		t.Fatalf("data len = %d, want 2", len(got.Data))
	}
	if got.Data[0].ID != "gpt-4o" || got.Data[0].Object != "model" || got.Data[0].OwnedBy != "openai" {
		t.Errorf("data[0] = %+v, want gpt-4o/model/openai", got.Data[0])
	}
	if got.Data[0].Created != 1715000000 {
		t.Errorf("data[0].created = %d, want 1715000000", got.Data[0].Created)
	}
}

func TestResponsesParser_ParseStreamUsage_CRLF(t *testing.T) {
	body := []byte("data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_1\",\"model\":\"gpt-4o\"}}\r\n\r\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"model\":\"gpt-4o\",\"usage\":{\"input_tokens\":7,\"output_tokens\":3,\"input_tokens_details\":{\"cached_tokens\":2}}}}\r\n\r\n")
	usage, err := Responses.ParseStreamUsage(body)
	if err != nil {
		t.Fatalf("ParseStreamUsage error: %v", err)
	}
	if usage == nil {
		t.Fatal("usage is nil")
	}
	if usage.Model != "gpt-4o" || usage.RequestID != "resp_1" {
		t.Fatalf("identity = model:%q request:%q", usage.Model, usage.RequestID)
	}
	if usage.InputTokens != 7 || usage.OutputTokens != 3 || usage.TotalTokens != 10 || usage.CachedTokens != 2 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
}
