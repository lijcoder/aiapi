package parser

import (
	"testing"
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

func TestResponsesParser_ParseStreamEvent(t *testing.T) {
	p := &ResponsesParser{}

	t.Run("response.created", func(t *testing.T) {
		evt, err := p.ParseStreamEvent([]byte(`{"type":"response.created","response":{"id":"resp_1","model":"gpt-4o"}}`))
		if err != nil {
			t.Fatalf("ParseStreamEvent error: %v", err)
		}
		if evt == nil {
			t.Fatal("ParseStreamEvent = nil, want event")
		}
		if evt.EventType != "usage" {
			t.Errorf("EventType = %q, want usage", evt.EventType)
		}
		if evt.Usage == nil {
			t.Fatal("Usage = nil, want usage")
		}
		if evt.Usage.Model != "gpt-4o" || evt.Usage.RequestID != "resp_1" {
			t.Errorf("usage model/requestID = %q/%q, want gpt-4o/resp_1", evt.Usage.Model, evt.Usage.RequestID)
		}
	})

	t.Run("response.output_text.delta", func(t *testing.T) {
		evt, err := p.ParseStreamEvent([]byte(`{"type":"response.output_text.delta","delta":"Hello"}`))
		if err != nil {
			t.Fatalf("ParseStreamEvent error: %v", err)
		}
		if evt == nil || evt.EventType != "content" || evt.Content != "Hello" {
			t.Fatalf("event = %+v, want content/Hello", evt)
		}
	})

	t.Run("response.output_text.delta empty", func(t *testing.T) {
		evt, err := p.ParseStreamEvent([]byte(`{"type":"response.output_text.delta","delta":""}`))
		if err != nil {
			t.Fatalf("ParseStreamEvent error: %v", err)
		}
		if evt != nil {
			t.Errorf("event = %+v, want nil for empty delta", evt)
		}
	})

	t.Run("response.completed", func(t *testing.T) {
		evt, err := p.ParseStreamEvent([]byte(`{
			"type":"response.completed",
			"response":{
				"id":"resp_2","model":"gpt-4o",
				"usage":{"input_tokens":200,"output_tokens":80,"total_tokens":280,
					"input_tokens_details":{"cached_tokens":40},
					"output_tokens_details":{"reasoning_tokens":5}}
			}
		}`))
		if err != nil {
			t.Fatalf("ParseStreamEvent error: %v", err)
		}
		if evt == nil || evt.EventType != "usage" {
			t.Fatalf("event = %+v, want usage", evt)
		}
		u := evt.Usage
		if u.InputTokens != 200 || u.OutputTokens != 80 || u.TotalTokens != 280 {
			t.Errorf("tokens = %d/%d/%d, want 200/80/280", u.InputTokens, u.OutputTokens, u.TotalTokens)
		}
		if u.CachedTokens != 40 || u.ReasoningTokens != 5 {
			t.Errorf("details = cached %d / reasoning %d, want 40/5", u.CachedTokens, u.ReasoningTokens)
		}
		if u.RequestID != "resp_2" {
			t.Errorf("RequestID = %q, want resp_2", u.RequestID)
		}
	})

	t.Run("response.failed", func(t *testing.T) {
		evt, err := p.ParseStreamEvent([]byte(`{"type":"response.failed","response":{"id":"resp_3","status":"failed"}}`))
		if err != nil {
			t.Fatalf("ParseStreamEvent error: %v", err)
		}
		if evt == nil || evt.EventType != "done" {
			t.Fatalf("event = %+v, want done", evt)
		}
	})

	t.Run("intermediate event skipped", func(t *testing.T) {
		for _, data := range []string{
			`{"type":"response.in_progress","response":{"id":"resp_1"}}`,
			`{"type":"response.output_text.done","text":"Hello"}`,
			`{"type":"response.output_item.added","output_index":0}`,
			`{"type":"response.content_part.done","content_index":0}`,
		} {
			evt, err := p.ParseStreamEvent([]byte(data))
			if err != nil {
				t.Fatalf("ParseStreamEvent(%q) error: %v", data, err)
			}
			if evt != nil {
				t.Errorf("ParseStreamEvent(%q) = %+v, want nil", data, evt)
			}
		}
	})
}
