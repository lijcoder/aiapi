package parser

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/lijcoder/aiapi/parser/util"
)

// Anthropic 协议的 golden 测试：fixture 见 testdata/anthropic_*.{json,sse}，
// 共用辅助函数（readFixture / goldenUsage / assertUsage 等）定义在 golden_openai_test.go。

// TestGoldenAnthropic_ParseModel 从请求体顶层 model 取模型名。
func TestGoldenAnthropic_ParseModel(t *testing.T) {
	cases := []struct {
		name string
		file string
		body string
		want string
	}{
		{"真实 messages 请求体", "anthropic_messages_request.json", "", "claude-sonnet-4-20250514"},
		{"model 不是字符串", "anthropic_messages_request_model_not_string.json", "", ""},
		{"错误响应体没有 model", "anthropic_error_response.json", "", ""},
		{"非法 JSON", "anthropic_invalid.json", "", ""},
		{"空 body", "", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := fixtureOrBody(t, tc.file, tc.body)
			if got := Anthropic.ParseModel(body); got != tc.want {
				t.Fatalf("ParseModel = %q, 期望 %q", got, tc.want)
			}
		})
	}
}

// TestGoldenAnthropic_ReplaceModel 改写只动顶层 model；无需改写时原样返回入参。
func TestGoldenAnthropic_ReplaceModel(t *testing.T) {
	t.Run("改写 fixture 中的模型名", func(t *testing.T) {
		body := readFixture(t, "anthropic_messages_request.json")
		out, err := Anthropic.ReplaceModel(body, "claude-haiku-4-20250514")
		if err != nil {
			t.Fatalf("ReplaceModel error: %v", err)
		}
		if got := Anthropic.ParseModel(out); got != "claude-haiku-4-20250514" {
			t.Fatalf("改写后 model = %q, 期望 claude-haiku-4-20250514", got)
		}
		if bytes.Equal(body, out) {
			t.Fatal("model 已改变却未改写 body")
		}
		// system / messages / max_tokens 等其余字段语义不变
		assertJSONEqualExcept(t, body, out, "model")
	})

	t.Run("无需改写时原样返回入参", func(t *testing.T) {
		cases := []struct {
			name   string
			file   string
			body   string
			target string
		}{
			{"model 已等于目标值", "anthropic_messages_request.json", "", "claude-sonnet-4-20250514"},
			{"顶层没有 model 字段", "anthropic_count_tokens_response.json", "", "claude-haiku-4-20250514"},
			{"非法 JSON 定位不到 model", "anthropic_invalid.json", "", "claude-haiku-4-20250514"},
			{"空 body", "", "", "claude-haiku-4-20250514"},
			{"JSON null 不是对象", "", `null`, "claude-haiku-4-20250514"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				body := fixtureOrBody(t, tc.file, tc.body)
				out, err := Anthropic.ReplaceModel(body, tc.target)
				if err != nil {
					t.Fatalf("ReplaceModel error: %v", err)
				}
				assertSameBody(t, body, out)
			})
		}
	})

	t.Run("model 不是字符串时报错", func(t *testing.T) {
		body := readFixture(t, "anthropic_messages_request_model_not_string.json")
		out, err := Anthropic.ReplaceModel(body, "claude-haiku-4-20250514")
		if !errors.Is(err, util.ErrModelNotString) {
			t.Fatalf("err = %v, 期望 util.ErrModelNotString", err)
		}
		if out != nil {
			t.Fatalf("出错时 body = %q, 期望 nil", out)
		}
	})
}

// TestGoldenAnthropic_ParseApiKey 鉴权头优先级：x-api-key → Authorization；两者都会剥 Bearer 前缀，裸 token 原样返回。
func TestGoldenAnthropic_ParseApiKey(t *testing.T) {
	cases := []struct {
		name    string
		headers map[string][]string
		want    string
	}{
		{"x-api-key 优先", map[string][]string{"X-Api-Key": {"sk-ant-golden"}, HeaderAuthorization: {"Bearer sk-bearer"}}, "sk-ant-golden"},
		{"x-api-key 为裸 token 原样返回", map[string][]string{"X-Api-Key": {"sk-ant-golden"}}, "sk-ant-golden"},
		{"x-api-key 带 Bearer 前缀也剥掉", map[string][]string{"X-Api-Key": {"Bearer sk-ant-golden"}}, "sk-ant-golden"},
		{"头名大小写不敏感（net/http 会规范化）", map[string][]string{"x-api-key": {"sk-ant-golden"}}, "sk-ant-golden"},
		{"x-api-key 为空回退 Authorization", map[string][]string{"X-Api-Key": {""}, HeaderAuthorization: {"Bearer sk-bearer"}}, "sk-bearer"},
		{"只传 Authorization 时用它", map[string][]string{HeaderAuthorization: {"Bearer sk-bearer"}}, "sk-bearer"},
		{"Authorization 裸 token 原样返回", map[string][]string{HeaderAuthorization: {"sk-raw"}}, "sk-raw"},
		{"Bearer 前缀大小写不敏感", map[string][]string{HeaderAuthorization: {"bEaReR sk-bearer"}}, "sk-bearer"},
		{"两者都没有", map[string][]string{}, ""},
		{"x-api-key 为空切片仍回退 Authorization", map[string][]string{"X-Api-Key": {}, HeaderAuthorization: {"Bearer sk-bearer"}}, "sk-bearer"},
		{"值恰好是 Bearer 加空格时前缀不剥（长度必须大于 7）", map[string][]string{"X-Api-Key": {"Bearer "}}, "Bearer "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Anthropic.ParseApiKey(tc.headers); got != tc.want {
				t.Fatalf("ParseApiKey = %q, 期望 %q", got, tc.want)
			}
		})
	}
}

// TestGoldenAnthropic_ParseUsage 非流式用量：完整输入 = input + 缓存创建 + 缓存读取，
// cached_tokens 取缓存读取，reasoning_tokens 取 output_tokens_details.thinking_tokens，total = 完整输入 + 输出。
func TestGoldenAnthropic_ParseUsage(t *testing.T) {
	cases := []struct {
		name    string
		file    string
		want    goldenUsage
		wantNil bool
	}{
		{
			name: "标准 message 响应（含缓存创建/读取与思考 token）",
			file: "anthropic_messages_response.json",
			want: goldenUsage{
				Provider: FormatAnthropic, Model: "claude-sonnet-4-20250514", RequestID: "msg_01Xy8kQ2rT7vW4zB6nM3pL9d",
				InputTokens: 9528, OutputTokens: 486, TotalTokens: 10014, CachedTokens: 8192, ReasoningTokens: 210,
			},
		},
		{
			// 有 type=message 但没有 usage：返回非 nil 的零用量（identity 仍被填上）
			name: "usage 缺失时返回非 nil 的零用量",
			file: "anthropic_messages_response_usage_missing.json",
			want: goldenUsage{
				Provider: FormatAnthropic, Model: "claude-sonnet-4-20250514", RequestID: "msg_01AbC2dE3fG4hI5jK6lM7nO9",
			},
		},
		{
			name:    "count_tokens 响应（type 不是 message）不产生用量",
			file:    "anthropic_count_tokens_response.json",
			wantNil: true,
		},
		{
			name:    "错误响应（type=error）不产生用量",
			file:    "anthropic_error_response.json",
			wantNil: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u, err := Anthropic.ParseUsage(readFixture(t, tc.file))
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

// TestGoldenAnthropic_ParseUsage_InvalidJSON 非法 JSON 返回错误与 nil 用量。
func TestGoldenAnthropic_ParseUsage_InvalidJSON(t *testing.T) {
	u, err := Anthropic.ParseUsage(readFixture(t, "anthropic_invalid.json"))
	if err == nil {
		t.Fatal("非法 JSON 未返回错误")
	}
	if u != nil {
		t.Fatalf("用量 = %+v, 期望 nil", u)
	}
}

// TestGoldenAnthropic_ParseStreamUsage 流式用量：message_delta 给累计用量，
// message_start 只给 identity（其 output_tokens=1 不得参与用量）。
func TestGoldenAnthropic_ParseStreamUsage(t *testing.T) {
	cases := []struct {
		name string
		file string
		want goldenUsage
	}{
		{
			name: "标准 SSE（message_start + message_delta + message_stop）",
			file: "anthropic_messages_stream.sse",
			want: goldenUsage{
				Provider: FormatAnthropic, Model: "claude-sonnet-4-20250514", RequestID: "msg_01Xy8kQ2rT7vW4zB6nM3pL9d",
				InputTokens: 9528, OutputTokens: 486, TotalTokens: 10014, CachedTokens: 8192, ReasoningTokens: 210,
			},
		},
		{
			// 老形态上游（message_delta 只回 output_tokens）：输入侧记 0，identity 仍来自 message_start
			name: "message_delta 只带 output_tokens 时输入量记 0",
			file: "anthropic_messages_stream_delta_output_only.sse",
			want: goldenUsage{
				Provider: FormatAnthropic, Model: "claude-sonnet-4-20250514", RequestID: "msg_01Rs4Tu5Vw6Xy7Za8Bc9De0f",
				InputTokens: 0, OutputTokens: 486, TotalTokens: 486,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u, err := Anthropic.ParseStreamUsage(readFixture(t, tc.file))
			if err != nil {
				t.Fatalf("ParseStreamUsage error: %v", err)
			}
			assertUsage(t, u, tc.want)
		})
	}
}

// TestGoldenAnthropic_ParseStreamUsage_NoMessageDelta 流式响应没有 message_delta（上游异常截断）时不记用量。
func TestGoldenAnthropic_ParseStreamUsage_NoMessageDelta(t *testing.T) {
	u, err := Anthropic.ParseStreamUsage(readFixture(t, "anthropic_messages_stream_no_delta.sse"))
	if err != nil {
		t.Fatalf("ParseStreamUsage error: %v", err)
	}
	if u != nil {
		t.Fatalf("用量 = %+v, 期望 nil（无 message_delta 不记零用量）", u)
	}
}

// TestGoldenAnthropic_ParseStreamUsage_MultilineData 单个 SSE 事件用多行 data: 承载一个 JSON 时，
// 各 data 行被当成独立载荷解析：JSON 被拆断后无法解析，用量为 nil。
// （SSE 规范要求多行 data 按 \n 拼接，这里是当前实现的已知边界。）
func TestGoldenAnthropic_ParseStreamUsage_MultilineData(t *testing.T) {
	u, err := Anthropic.ParseStreamUsage(readFixture(t, "anthropic_messages_stream_multiline_data.sse"))
	if err != nil {
		t.Fatalf("ParseStreamUsage error: %v", err)
	}
	if u != nil {
		t.Fatalf("用量 = %+v, 期望 nil（多行 data 被逐行解析，无法还原事件）", u)
	}
}

// TestGoldenAnthropic_ParseStreamUsage_InvalidSSE 非 SSE 内容不报错，只是没有用量。
func TestGoldenAnthropic_ParseStreamUsage_InvalidSSE(t *testing.T) {
	u, err := Anthropic.ParseStreamUsage(readFixture(t, "anthropic_invalid.json"))
	if err != nil {
		t.Fatalf("ParseStreamUsage error: %v", err)
	}
	if u != nil {
		t.Fatalf("用量 = %+v, 期望 nil", u)
	}
}

// TestGoldenAnthropic_FormatModelList Anthropic 模型列表：
// {data:[{type,id,display_name,created_at}], first_id, last_id, has_more}。
func TestGoldenAnthropic_FormatModelList(t *testing.T) {
	// 第二个模型用非 UTC 时区构造，验证 created_at 会转成 UTC 的 RFC3339
	items := []ModelItem{
		{ID: "claude-sonnet-4-20250514", CreatedAt: time.Unix(1715000000, 0).In(time.FixedZone("CST", 8*3600))},
		{ID: "claude-haiku-4-20250514", CreatedAt: time.Unix(1715000100, 0)},
	}
	body, err := FormatModelList(Anthropic, items)
	if err != nil {
		t.Fatalf("FormatModelList error: %v", err)
	}

	var top map[string]json.RawMessage
	if err := json.Unmarshal(body, &top); err != nil {
		t.Fatalf("解析模型列表响应失败: %v", err)
	}
	if len(top) != 4 || top["data"] == nil || top["first_id"] == nil || top["last_id"] == nil || top["has_more"] == nil {
		t.Fatalf("顶层字段 = %v, 期望恰好是 data / first_id / last_id / has_more", keysOf(top))
	}
	if got := string(top["has_more"]); got != "false" {
		t.Fatalf("has_more = %s, 期望 false（本地全量返回）", got)
	}
	if got := string(top["first_id"]); got != `"claude-sonnet-4-20250514"` {
		t.Fatalf("first_id = %s, 期望首个模型 ID", got)
	}
	if got := string(top["last_id"]); got != `"claude-haiku-4-20250514"` {
		t.Fatalf("last_id = %s, 期望末个模型 ID", got)
	}

	var data []map[string]json.RawMessage
	if err := json.Unmarshal(top["data"], &data); err != nil {
		t.Fatalf("data 不是对象数组: %v", err)
	}
	wantCreatedAt := []string{"2024-05-06T12:53:20Z", "2024-05-06T12:55:00Z"}
	if len(data) != len(items) {
		t.Fatalf("data 长度 = %d, 期望 %d", len(data), len(items))
	}
	for i, item := range items {
		entry := data[i]
		if len(entry) != 4 || entry["type"] == nil || entry["id"] == nil || entry["display_name"] == nil || entry["created_at"] == nil {
			t.Fatalf("data[%d] 字段 = %v, 期望恰好是 type / id / display_name / created_at", i, keysOf(entry))
		}
		entryBody, err := json.Marshal(entry)
		if err != nil {
			t.Fatalf("重新序列化 data[%d] 失败: %v", i, err)
		}
		var got struct {
			Type        string `json:"type"`
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
			CreatedAt   string `json:"created_at"`
		}
		if err := json.Unmarshal(entryBody, &got); err != nil {
			t.Fatalf("data[%d] 结构不符: %v", i, err)
		}
		if got.Type != "model" || got.ID != item.ID || got.DisplayName != item.ID || got.CreatedAt != wantCreatedAt[i] {
			t.Fatalf("data[%d] = %+v, 期望 type=model id=%s display_name=%s created_at=%s",
				i, got, item.ID, item.ID, wantCreatedAt[i])
		}
	}
}

// TestGoldenAnthropic_FormatModelList_Empty 空列表：data 为 []，first_id / last_id 为 null，has_more 为 false。
func TestGoldenAnthropic_FormatModelList_Empty(t *testing.T) {
	body, err := FormatModelList(Anthropic, nil)
	if err != nil {
		t.Fatalf("FormatModelList error: %v", err)
	}
	want := `{"data":[],"first_id":null,"last_id":null,"has_more":false}`
	if got := string(body); got != want {
		t.Fatalf("空列表响应 = %s, 期望 %s", got, want)
	}
}
