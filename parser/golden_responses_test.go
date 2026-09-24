package parser

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/lijcoder/aiapi/parser/util"
)

// Responses API 协议的 golden 测试：fixture 见 testdata/responses_*.{json,sse}，
// 共用辅助函数（readFixture / goldenUsage / assertUsage 等）定义在 golden_openai_test.go。

// TestGoldenResponses_ParseModel 从请求体顶层 model 取模型名。
func TestGoldenResponses_ParseModel(t *testing.T) {
	cases := []struct {
		name string
		file string
		body string
		want string
	}{
		{"真实 responses 请求体", "responses_request.json", "", "gpt-4o-2024-08-06"},
		{"model 不是字符串", "responses_request_model_not_string.json", "", ""},
		{"错误响应体没有 model", "responses_error_response.json", "", ""},
		{"非法 JSON", "responses_invalid.json", "", ""},
		{"空 body", "", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := fixtureOrBody(t, tc.file, tc.body)
			if got := Responses.ParseModel(body); got != tc.want {
				t.Fatalf("ParseModel = %q, 期望 %q", got, tc.want)
			}
		})
	}
}

// TestGoldenResponses_ReplaceModel 改写只动顶层 model；无需改写时原样返回入参。
func TestGoldenResponses_ReplaceModel(t *testing.T) {
	t.Run("改写 fixture 中的模型名", func(t *testing.T) {
		body := readFixture(t, "responses_request.json")
		out, err := Responses.ReplaceModel(body, "gpt-4o-mini")
		if err != nil {
			t.Fatalf("ReplaceModel error: %v", err)
		}
		if got := Responses.ParseModel(out); got != "gpt-4o-mini" {
			t.Fatalf("改写后 model = %q, 期望 gpt-4o-mini", got)
		}
		if bytes.Equal(body, out) {
			t.Fatal("model 已改变却未改写 body")
		}
		// input / instructions / max_output_tokens 等其余字段语义不变
		assertJSONEqualExcept(t, body, out, "model")
	})

	t.Run("无需改写时原样返回入参", func(t *testing.T) {
		cases := []struct {
			name   string
			file   string
			body   string
			target string
		}{
			{"model 已等于目标值", "responses_request.json", "", "gpt-4o-2024-08-06"},
			{"顶层没有 model 字段", "responses_error_response.json", "", "gpt-4o-mini"},
			{"非法 JSON 定位不到 model", "responses_invalid.json", "", "gpt-4o-mini"},
			{"空 body", "", "", "gpt-4o-mini"},
			{"JSON 数字不是对象", "", `123`, "gpt-4o-mini"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				body := fixtureOrBody(t, tc.file, tc.body)
				out, err := Responses.ReplaceModel(body, tc.target)
				if err != nil {
					t.Fatalf("ReplaceModel error: %v", err)
				}
				assertSameBody(t, body, out)
			})
		}
	})

	t.Run("model 不是字符串时报错", func(t *testing.T) {
		body := readFixture(t, "responses_request_model_not_string.json")
		out, err := Responses.ReplaceModel(body, "gpt-4o-mini")
		if !errors.Is(err, util.ErrModelNotString) {
			t.Fatalf("err = %v, 期望 util.ErrModelNotString", err)
		}
		if out != nil {
			t.Fatalf("出错时 body = %q, 期望 nil", out)
		}
	})
}

// TestGoldenResponses_ParseApiKey 与 chat/completions 一致：只认 Authorization，Bearer 前缀剥掉，裸 token 原样返回。
func TestGoldenResponses_ParseApiKey(t *testing.T) {
	cases := []struct {
		name    string
		headers map[string][]string
		want    string
	}{
		{"Bearer 前缀被剥掉", map[string][]string{HeaderAuthorization: {"Bearer sk-golden-responses"}}, "sk-golden-responses"},
		{"Bearer 前缀大小写不敏感", map[string][]string{"authorization": {"bearer sk-golden-responses"}}, "sk-golden-responses"},
		{"裸 token 原样返回", map[string][]string{HeaderAuthorization: {"sk-golden-responses"}}, "sk-golden-responses"},
		{"x-api-key 不参与取值", map[string][]string{"X-Api-Key": {"sk-ignored"}}, ""},
		{"没有鉴权头", map[string][]string{}, ""},
		{"只取头部的第一个值", map[string][]string{HeaderAuthorization: {"", "Bearer sk-second"}}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Responses.ParseApiKey(tc.headers); got != tc.want {
				t.Fatalf("ParseApiKey = %q, 期望 %q", got, tc.want)
			}
		})
	}
}

// TestGoldenResponses_ParseUsage 非流式用量：字段名是 input_tokens / output_tokens（不是 chat 的 prompt/completion），
// 缓存与推理取 input_tokens_details.cached_tokens / output_tokens_details.reasoning_tokens。
func TestGoldenResponses_ParseUsage(t *testing.T) {
	cases := []struct {
		name    string
		file    string
		want    goldenUsage
		wantNil bool
	}{
		{
			name: "标准非流式响应",
			file: "responses_response.json",
			want: goldenUsage{
				Provider: FormatResponses, Model: "gpt-4o-2024-08-06", RequestID: "resp_67c9a1B2c3D4e5F6a7B8c9D0",
				InputTokens: 2048, OutputTokens: 512, TotalTokens: 2560, CachedTokens: 1536, ReasoningTokens: 96,
			},
		},
		{
			// object=response 但没有 usage：返回非 nil 的零用量（identity 仍被填上）
			name: "usage 缺失时返回非 nil 的零用量",
			file: "responses_response_usage_missing.json",
			want: goldenUsage{
				Provider: FormatResponses, Model: "gpt-4o-2024-08-06", RequestID: "resp_67c9a1B2c3D4e5F6a7B8c9D1",
			},
		},
		{
			// 非流式同样回退：上游省略 total_tokens 时按 input + output 记
			name: "上游省略 total_tokens 时回退 input + output",
			file: "responses_response_total_missing.json",
			want: goldenUsage{
				Provider: FormatResponses, Model: "gpt-4o-2024-08-06", RequestID: "resp_67c9a1B2c3D4e5F6a7B8c9D2",
				InputTokens: 640, OutputTokens: 160, TotalTokens: 800,
			},
		},
		{
			name:    "错误响应（object 不是 response）不产生用量",
			file:    "responses_error_response.json",
			wantNil: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u, err := Responses.ParseUsage(readFixture(t, tc.file))
			if err != nil {
				t.Fatalf("ParseUsage error: %v", err)
			}
			if tc.wantNil {
				if u != nil {
					t.Fatalf("用量 = %+v, 期望 nil", u)
				}
				return
			}
			assertUsage(t, u, tc.want)
		})
	}
}

// TestGoldenResponses_ParseUsage_InvalidJSON 非法 JSON 返回错误与 nil 用量。
func TestGoldenResponses_ParseUsage_InvalidJSON(t *testing.T) {
	u, err := Responses.ParseUsage(readFixture(t, "responses_invalid.json"))
	if err == nil {
		t.Fatal("非法 JSON 未返回错误")
	}
	if u != nil {
		t.Fatalf("用量 = %+v, 期望 nil", u)
	}
}

// TestGoldenResponses_ParseStreamUsage 流式用量只看 response.completed：
// 其余事件（created / in_progress / output_text.delta / *_done）即使带 response 对象也不参与。
func TestGoldenResponses_ParseStreamUsage(t *testing.T) {
	cases := []struct {
		name string
		file string
		want goldenUsage
	}{
		{
			name: "标准 SSE（created → delta → completed）",
			file: "responses_stream.sse",
			want: goldenUsage{
				Provider: FormatResponses, Model: "gpt-4o-2024-08-06", RequestID: "resp_67c9a1B2c3D4e5F6a7B8c9E0",
				InputTokens: 4096, OutputTokens: 1024, TotalTokens: 5120, CachedTokens: 3072, ReasoningTokens: 256,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u, err := Responses.ParseStreamUsage(readFixture(t, tc.file))
			if err != nil {
				t.Fatalf("ParseStreamUsage error: %v", err)
			}
			assertUsage(t, u, tc.want)
		})
	}
}

// TestGoldenResponses_ParseStreamUsage_NoCompleted 流式响应没有 response.completed（上游失败中断）时不记用量。
func TestGoldenResponses_ParseStreamUsage_NoCompleted(t *testing.T) {
	u, err := Responses.ParseStreamUsage(readFixture(t, "responses_stream_incomplete.sse"))
	if err != nil {
		t.Fatalf("ParseStreamUsage error: %v", err)
	}
	if u != nil {
		t.Fatalf("用量 = %+v, 期望 nil（无 response.completed 不记零用量）", u)
	}
}

// TestGoldenResponses_ParseStreamUsage_InvalidSSE 非 SSE 内容不报错，只是没有用量。
func TestGoldenResponses_ParseStreamUsage_InvalidSSE(t *testing.T) {
	u, err := Responses.ParseStreamUsage(readFixture(t, "responses_invalid.json"))
	if err != nil {
		t.Fatalf("ParseStreamUsage error: %v", err)
	}
	if u != nil {
		t.Fatalf("用量 = %+v, 期望 nil", u)
	}
}

// TestGoldenResponses_FormatModelList responses 的模型列表沿用 OpenAI List Models 形状
// （结构独立建模，字段名与 OpenAI 相同）。
func TestGoldenResponses_FormatModelList(t *testing.T) {
	items := []ModelItem{
		{ID: "gpt-4o-2024-08-06", CreatedAt: time.Unix(1715000000, 0).UTC(), OwnedBy: "openai"},
		{ID: "o4-mini", CreatedAt: time.Unix(1715000100, 0).UTC(), OwnedBy: "openai"},
	}
	body, err := FormatModelList(Responses, items)
	if err != nil {
		t.Fatalf("FormatModelList error: %v", err)
	}
	assertOpenAIListShape(t, body, items)
}
