package parser

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

// modelParsers 覆盖全部已实现协议的模型名改写（三者模型名都在请求体顶层）
var modelParsers = []struct {
	name string
	p    Parser
}{
	{"openai", OpenAI},
	{"anthropic", Anthropic},
	{"responses", Responses},
}

// TestReplaceModel_RewritesTopLevelModel 改写后顶层 model 生效，其余字段原样保留
func TestReplaceModel_RewritesTopLevelModel(t *testing.T) {
	body := []byte(`{"model":"gpt-4o","stream":true,"temperature":0.7,"max_tokens":10000000000000000001,"messages":[{"role":"user","content":"hello"}]}`)
	for _, tc := range modelParsers {
		t.Run(tc.name, func(t *testing.T) {
			out, err := tc.p.ReplaceModel(body, "gpt-4o-2024-08-06")
			if err != nil {
				t.Fatalf("ReplaceModel error: %v", err)
			}
			if got := tc.p.ParseModel(out); got != "gpt-4o-2024-08-06" {
				t.Fatalf("model = %q, want gpt-4o-2024-08-06", got)
			}
			// 其余字段按原始字节透传：数字不丢精度，嵌套结构不被重排
			src, dst := rawFields(t, body), rawFields(t, out)
			if len(src) != len(dst) {
				t.Fatalf("field count = %d, want %d", len(dst), len(src))
			}
			for key, want := range src {
				if key == "model" {
					continue
				}
				if got, ok := dst[key]; !ok || !bytes.Equal(got, want) {
					t.Fatalf("field %q = %s, want %s", key, got, want)
				}
			}
		})
	}
}

// TestReplaceModel_SemanticallyPreservesEscapedContent 改写会重新编码，但含 < > & 的内容语义不变
func TestReplaceModel_SemanticallyPreservesEscapedContent(t *testing.T) {
	body := []byte(`{"model":"a","messages":[{"content":"<b>x</b> & <i>y</i>"}]}`)
	for _, tc := range modelParsers {
		t.Run(tc.name, func(t *testing.T) {
			out, err := tc.p.ReplaceModel(body, "b")
			if err != nil {
				t.Fatalf("ReplaceModel error: %v", err)
			}
			if !jsonEqual(t, body, out, "messages") {
				t.Fatalf("messages changed: %s", out)
			}
		})
	}
}

// TestReplaceModel_NoRewriteKeepsOriginalBytes 无需改写时必须原样返回入参（零拷贝、字节零变动）
func TestReplaceModel_NoRewriteKeepsOriginalBytes(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"equals target", `{"model":"gpt-4o","stream":true}`, "gpt-4o"},
		{"no model field", `{"prompt":"hi"}`, "gpt-4o"},
		{"empty body", ``, "gpt-4o"},
		{"not json", `model=gpt-4o`, "gpt-4o"},
		{"json array", `[{"model":"gpt-4o"}]`, "gpt-4o"},
		{"json null", `null`, "gpt-4o"},
	}
	for _, tc := range modelParsers {
		for _, c := range cases {
			t.Run(tc.name+"/"+c.name, func(t *testing.T) {
				body := []byte(c.body)
				out, err := tc.p.ReplaceModel(body, c.want)
				if err != nil {
					t.Fatalf("ReplaceModel error: %v", err)
				}
				if len(out) != len(body) || (len(body) > 0 && &out[0] != &body[0]) {
					t.Fatalf("body was rewritten: %q", out)
				}
			})
		}
	}
}

// TestReplaceModel_NonStringModel 顶层 model 不是字符串时报错，由调用方决定失败语义
func TestReplaceModel_NonStringModel(t *testing.T) {
	for _, tc := range modelParsers {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.p.ReplaceModel([]byte(`{"model":123}`), "gpt-4o"); !errors.Is(err, ErrModelNotString) {
				t.Fatalf("err = %v, want ErrModelNotString", err)
			}
		})
	}
}

// TestReplaceModel_NullModel null 视为未设置，可被替换为字符串
func TestReplaceModel_NullModel(t *testing.T) {
	for _, tc := range modelParsers {
		t.Run(tc.name, func(t *testing.T) {
			out, err := tc.p.ReplaceModel([]byte(`{"model":null,"stream":true}`), "gpt-4o")
			if err != nil {
				t.Fatalf("ReplaceModel error: %v", err)
			}
			if got := tc.p.ParseModel(out); got != "gpt-4o" {
				t.Fatalf("model = %q, want gpt-4o", got)
			}
		})
	}
}

// rawFields 解析出顶层各字段的原始 JSON 片段
func rawFields(t *testing.T, body []byte) map[string]json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		t.Fatalf("unmarshal fields: %v", err)
	}
	return fields
}

// jsonEqual 比较两份 body 中指定字段的 JSON 语义是否一致
func jsonEqual(t *testing.T, a, b []byte, key string) bool {
	t.Helper()
	var left, right any
	if err := json.Unmarshal(rawFields(t, a)[key], &left); err != nil {
		t.Fatalf("unmarshal left %s: %v", key, err)
	}
	if err := json.Unmarshal(rawFields(t, b)[key], &right); err != nil {
		t.Fatalf("unmarshal right %s: %v", key, err)
	}
	return reflect.DeepEqual(left, right)
}
