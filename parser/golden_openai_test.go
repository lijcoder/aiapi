package parser

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/lijcoder/aiapi/parser/util"
)

// 本文件与 golden_anthropic_test.go / golden_responses_test.go 属于同一套 golden 测试：
// 三个协议各自的输入/输出 fixture 放在 testdata/ 下并按协议前缀命名，用 os.ReadFile 读取；
// 跨协议共用的辅助函数（读取 fixture、用量断言、body 语义比较等）集中定义在本文件里。

// readFixture 读取 testdata 下的 golden 文件。
// 断言的输入就是真实上游形状的原始字节，不在测试里重新拼装响应。
func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("读取 fixture %s 失败: %v", name, err)
	}
	return body
}

// fixtureOrBody file 非空时读 fixture，否则用极小内联 body（仅用于空 body 这类无法落文件的场景）。
func fixtureOrBody(t *testing.T, file, body string) []byte {
	t.Helper()
	if file != "" {
		return readFixture(t, file)
	}
	return []byte(body)
}

// goldenUsage 期望的统一用量：包含 Usage 的全部字段，便于逐字段断言。
type goldenUsage struct {
	Provider        string
	Model           string
	RequestID       string
	InputTokens     int
	OutputTokens    int
	TotalTokens     int
	CachedTokens    int
	ReasoningTokens int
}

// assertUsage 逐字段比对用量；nil 或任一字段不符都失败。
func assertUsage(t *testing.T, got *Usage, want goldenUsage) {
	t.Helper()
	if got == nil {
		t.Fatalf("用量为 nil, 期望 %+v", want)
	}
	if got.Provider != want.Provider {
		t.Errorf("Provider = %q, 期望 %q", got.Provider, want.Provider)
	}
	if got.Model != want.Model {
		t.Errorf("Model = %q, 期望 %q", got.Model, want.Model)
	}
	if got.RequestID != want.RequestID {
		t.Errorf("RequestID = %q, 期望 %q", got.RequestID, want.RequestID)
	}
	if got.InputTokens != want.InputTokens {
		t.Errorf("InputTokens = %d, 期望 %d", got.InputTokens, want.InputTokens)
	}
	if got.OutputTokens != want.OutputTokens {
		t.Errorf("OutputTokens = %d, 期望 %d", got.OutputTokens, want.OutputTokens)
	}
	if got.TotalTokens != want.TotalTokens {
		t.Errorf("TotalTokens = %d, 期望 %d", got.TotalTokens, want.TotalTokens)
	}
	if got.CachedTokens != want.CachedTokens {
		t.Errorf("CachedTokens = %d, 期望 %d", got.CachedTokens, want.CachedTokens)
	}
	if got.ReasoningTokens != want.ReasoningTokens {
		t.Errorf("ReasoningTokens = %d, 期望 %d", got.ReasoningTokens, want.ReasoningTokens)
	}
}

// assertSameBody 断言返回的 body 与入参是同一段字节：长度相同且底层数组未被重新分配，
// 即「无需改写时原样返回入参 body、字节零变动」。
func assertSameBody(t *testing.T, in, out []byte) {
	t.Helper()
	if len(in) != len(out) {
		t.Fatalf("body 长度 = %d, 期望 %d（无需改写时应原样返回入参）", len(out), len(in))
	}
	if len(in) > 0 && &in[0] != &out[0] {
		t.Fatalf("body 被复制或重新编码, 期望原样返回入参（字节零变动）")
	}
}

// assertJSONEqualExcept 断言两份 body 除 skip 指定的顶层字段外 JSON 语义完全一致。
// 改写走的是 map[string]json.RawMessage 重新编码，键序与空白会变，因此按语义比较；
// 用 json.Number 解码以保留数字精度。
func assertJSONEqualExcept(t *testing.T, before, after []byte, skip string) {
	t.Helper()
	left, right := decodeJSONObject(t, before), decodeJSONObject(t, after)
	delete(left, skip)
	delete(right, skip)
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("除 %q 外的字段发生变化:\n改写前 %#v\n改写后 %#v", skip, left, right)
	}
}

// decodeJSONObject 把 JSON 对象解码为 map，数字保留原始字面量（json.Number）。
func decodeJSONObject(t *testing.T, body []byte) map[string]any {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var fields map[string]any
	if err := dec.Decode(&fields); err != nil {
		t.Fatalf("解析 JSON 失败: %v", err)
	}
	return fields
}

// keysOf 返回 map 的键集合（排序后），只用于失败信息。
func keysOf(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// nonFormatterParser 只实现 Parser、不实现 ModelsFormatter，
// 用于验证 FormatModelList 回退到 OpenAI 格式的分支。
type nonFormatterParser struct{}

func (nonFormatterParser) ParseModel([]byte) string                        { return "" }
func (nonFormatterParser) ReplaceModel(b []byte, _ string) ([]byte, error) { return b, nil }
func (nonFormatterParser) ParseApiKey(map[string][]string) string          { return "" }
func (nonFormatterParser) ParseUsage([]byte) (*Usage, error)               { return nil, nil }
func (nonFormatterParser) ParseStreamUsage(body []byte) (*Usage, error)    { return nil, nil }

// TestGoldenOpenAI_ParseModel 从请求体顶层 model 取模型名。
func TestGoldenOpenAI_ParseModel(t *testing.T) {
	cases := []struct {
		name string
		file string
		body string
		want string
	}{
		{"真实 chat/completions 请求体", "openai_chat_request.json", "", "gpt-4o-2024-08-06"},
		{"model 不是字符串", "openai_chat_request_model_not_string.json", "", ""},
		{"错误响应体没有 model", "openai_error_response.json", "", ""},
		{"非法 JSON", "openai_invalid.json", "", ""},
		{"空 body", "", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := fixtureOrBody(t, tc.file, tc.body)
			if got := OpenAI.ParseModel(body); got != tc.want {
				t.Fatalf("ParseModel = %q, 期望 %q", got, tc.want)
			}
		})
	}
}

// TestGoldenOpenAI_ReplaceModel 改写与不改写两条路径：
// 改写时只动顶层 model 且其余字段语义不变；无需改写时必须原样返回入参 body。
func TestGoldenOpenAI_ReplaceModel(t *testing.T) {
	t.Run("改写 fixture 中的模型名", func(t *testing.T) {
		body := readFixture(t, "openai_chat_request.json")
		out, err := OpenAI.ReplaceModel(body, "gpt-4o-mini")
		if err != nil {
			t.Fatalf("ReplaceModel error: %v", err)
		}
		if got := OpenAI.ParseModel(out); got != "gpt-4o-mini" {
			t.Fatalf("改写后 model = %q, 期望 gpt-4o-mini", got)
		}
		if bytes.Equal(body, out) {
			t.Fatal("model 已改变却未改写 body")
		}
		// messages / stream_options / temperature 等其余字段语义不变
		assertJSONEqualExcept(t, body, out, "model")
	})

	t.Run("无需改写时原样返回入参", func(t *testing.T) {
		cases := []struct {
			name   string
			file   string
			body   string
			target string
		}{
			{"model 已等于目标值", "openai_chat_request.json", "", "gpt-4o-2024-08-06"},
			{"顶层没有 model 字段", "openai_error_response.json", "", "gpt-4o-mini"},
			{"非法 JSON 定位不到 model", "openai_invalid.json", "", "gpt-4o-mini"},
			{"空 body", "", "", "gpt-4o-mini"},
			{"JSON 数组不是对象", "", `[{"model":"gpt-4o"}]`, "gpt-4o-mini"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				body := fixtureOrBody(t, tc.file, tc.body)
				out, err := OpenAI.ReplaceModel(body, tc.target)
				if err != nil {
					t.Fatalf("ReplaceModel error: %v", err)
				}
				assertSameBody(t, body, out)
			})
		}
	})

	t.Run("model 不是字符串时报错", func(t *testing.T) {
		body := readFixture(t, "openai_chat_request_model_not_string.json")
		out, err := OpenAI.ReplaceModel(body, "gpt-4o-mini")
		if !errors.Is(err, util.ErrModelNotString) {
			t.Fatalf("err = %v, 期望 util.ErrModelNotString", err)
		}
		if out != nil {
			t.Fatalf("出错时 body = %q, 期望 nil", out)
		}
	})
}

// TestGoldenOpenAI_ParseApiKey OpenAI 只认 Authorization：Bearer 前缀（大小写不敏感）剥掉，裸 token 原样返回。
func TestGoldenOpenAI_ParseApiKey(t *testing.T) {
	cases := []struct {
		name    string
		headers map[string][]string
		want    string
	}{
		{"Bearer 前缀被剥掉", map[string][]string{HeaderAuthorization: {"Bearer sk-golden-openai"}}, "sk-golden-openai"},
		{"Bearer 前缀大小写不敏感", map[string][]string{"authorization": {"bearer sk-golden-openai"}}, "sk-golden-openai"},
		{"裸 token 原样返回", map[string][]string{HeaderAuthorization: {"sk-golden-openai"}}, "sk-golden-openai"},
		{"x-api-key 不是 OpenAI 的鉴权头", map[string][]string{"X-Api-Key": {"sk-ignored"}}, ""},
		{"两个头同时存在时仍只看 Authorization", map[string][]string{"X-Api-Key": {"sk-ignored"}, HeaderAuthorization: {"Bearer sk-golden-openai"}}, "sk-golden-openai"},
		{"没有鉴权头", map[string][]string{}, ""},
		{"头部值为空切片", map[string][]string{HeaderAuthorization: {}}, ""},
		{"只取头部的第一个值", map[string][]string{HeaderAuthorization: {"", "Bearer sk-second"}}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := OpenAI.ParseApiKey(tc.headers); got != tc.want {
				t.Fatalf("ParseApiKey = %q, 期望 %q", got, tc.want)
			}
		})
	}
}

// TestGoldenOpenAI_ParseUsage 非流式用量：chat 的 prompt_tokens / completion_tokens，
// 缓存与推理分别取 *_tokens_details。
func TestGoldenOpenAI_ParseUsage(t *testing.T) {
	cases := []struct {
		name string
		file string
		want goldenUsage
	}{
		{
			name: "标准非流式响应",
			file: "openai_chat_response.json",
			want: goldenUsage{
				Provider: FormatOpenAI, Model: "gpt-4o-2024-08-06", RequestID: "chatcmpl-BnK9pQ2xR7sT4vW1yZ6aD3",
				InputTokens: 1234, OutputTokens: 567, TotalTokens: 1801, CachedTokens: 1024, ReasoningTokens: 128,
			},
		},
		{
			// 上游没回 usage 时仍返回非 nil 的零用量（调用方按非 nil 记账）
			name: "usage 缺失时返回非 nil 的零用量",
			file: "openai_chat_response_usage_missing.json",
			want: goldenUsage{
				Provider: FormatOpenAI, Model: "gpt-4o-2024-08-06", RequestID: "chatcmpl-BnK9pQ2xR7sT4vW1yZ6aD4",
			},
		},
		{
			// 非流式不做 total 回退：上游省略 total_tokens 时记 0（流式同场景会回退，见 ParseStreamUsage 用例）
			name: "上游省略 total_tokens 时非流式不回退",
			file: "openai_chat_response_total_missing.json",
			want: goldenUsage{
				Provider: FormatOpenAI, Model: "gpt-4o-2024-08-06", RequestID: "chatcmpl-BnK9pQ2xR7sT4vW1yZ6aD5",
				InputTokens: 512, OutputTokens: 128, TotalTokens: 0,
			},
		},
		{
			// 错误体也是合法 JSON 对象：没有 object 校验，被当成零用量
			name: "上游错误体被当作零用量",
			file: "openai_error_response.json",
			want: goldenUsage{Provider: FormatOpenAI},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u, err := OpenAI.ParseUsage(readFixture(t, tc.file))
			if err != nil {
				t.Fatalf("ParseUsage error: %v", err)
			}
			assertUsage(t, u, tc.want)
		})
	}
}

// TestGoldenOpenAI_ParseUsage_InvalidJSON 非法 JSON 返回错误与 nil 用量。
func TestGoldenOpenAI_ParseUsage_InvalidJSON(t *testing.T) {
	u, err := OpenAI.ParseUsage(readFixture(t, "openai_invalid.json"))
	if err == nil {
		t.Fatal("非法 JSON 未返回错误")
	}
	if u != nil {
		t.Fatalf("用量 = %+v, 期望 nil", u)
	}
}

// TestGoldenOpenAI_ParseStreamUsage 流式用量：只在带 usage 的块取值，
// identity（id / model）同样只从该块取（增量块的 id / model 不参与）。
func TestGoldenOpenAI_ParseStreamUsage(t *testing.T) {
	cases := []struct {
		name string
		file string
		want goldenUsage
	}{
		{
			name: "标准 SSE（含 usage 块与 [DONE]）",
			file: "openai_chat_stream.sse",
			want: goldenUsage{
				Provider: FormatOpenAI, Model: "gpt-4o-2024-08-06", RequestID: "chatcmpl-BnK9pQ2xR7sT4vW1yZ6aD6",
				InputTokens: 2048, OutputTokens: 256, TotalTokens: 2304, CachedTokens: 1536, ReasoningTokens: 64,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u, err := OpenAI.ParseStreamUsage(readFixture(t, tc.file))
			if err != nil {
				t.Fatalf("ParseStreamUsage error: %v", err)
			}
			assertUsage(t, u, tc.want)
		})
	}
}

// TestGoldenOpenAI_ParseStreamUsage_NoUsage 流式响应完全没有 usage 块（未开启 include_usage）时返回 nil, nil。
func TestGoldenOpenAI_ParseStreamUsage_NoUsage(t *testing.T) {
	u, err := OpenAI.ParseStreamUsage(readFixture(t, "openai_chat_stream_no_usage.sse"))
	if err != nil {
		t.Fatalf("ParseStreamUsage error: %v", err)
	}
	if u != nil {
		t.Fatalf("用量 = %+v, 期望 nil（无 usage 块不记零用量）", u)
	}
}

// TestGoldenOpenAI_ParseStreamUsage_TotalFallback 流式下上游省略 total_tokens 时回退 input + output。
func TestGoldenOpenAI_ParseStreamUsage_TotalFallback(t *testing.T) {
	u, err := OpenAI.ParseStreamUsage(readFixture(t, "openai_chat_stream_total_missing.sse"))
	if err != nil {
		t.Fatalf("ParseStreamUsage error: %v", err)
	}
	assertUsage(t, u, goldenUsage{
		Provider: FormatOpenAI, Model: "gpt-4o-2024-08-06", RequestID: "chatcmpl-BnK9pQ2xR7sT4vW1yZ6aD8",
		InputTokens: 321, OutputTokens: 123, TotalTokens: 444,
	})
}

// TestGoldenOpenAI_ParseStreamUsage_InvalidSSE 非法/非 SSE 内容不报错，只是没有用量。
func TestGoldenOpenAI_ParseStreamUsage_InvalidSSE(t *testing.T) {
	u, err := OpenAI.ParseStreamUsage(readFixture(t, "openai_invalid.json"))
	if err != nil {
		t.Fatalf("ParseStreamUsage error: %v", err)
	}
	if u != nil {
		t.Fatalf("用量 = %+v, 期望 nil", u)
	}
}

// TestGoldenOpenAI_FormatModelList OpenAI 模型列表：{object:"list", data:[{id,object,created,owned_by}]}。
func TestGoldenOpenAI_FormatModelList(t *testing.T) {
	items := []ModelItem{
		{ID: "gpt-4o-2024-08-06", CreatedAt: time.Unix(1715000000, 0).UTC(), OwnedBy: "openai"},
		{ID: "gpt-4o-mini", CreatedAt: time.Unix(1715000100, 0).UTC(), OwnedBy: "openai"},
	}
	body, err := FormatModelList(OpenAI, items)
	if err != nil {
		t.Fatalf("FormatModelList error: %v", err)
	}
	assertOpenAIListShape(t, body, items)
}

// TestGoldenOpenAI_FormatModelList_Empty 空列表也是合法响应：object 为 list、data 为 []（不是 null）。
func TestGoldenOpenAI_FormatModelList_Empty(t *testing.T) {
	body, err := FormatModelList(OpenAI, nil)
	if err != nil {
		t.Fatalf("FormatModelList error: %v", err)
	}
	if got := string(body); got != `{"object":"list","data":[]}` {
		t.Fatalf("空列表响应 = %s, 期望 {\"object\":\"list\",\"data\":[]}", got)
	}
}

// TestGoldenOpenAI_FormatModelList_Fallback 未实现 ModelsFormatter 的 Parser（含 nil）
// 回退到 OpenAI 格式，形状与 OpenAI 一致。
func TestGoldenOpenAI_FormatModelList_Fallback(t *testing.T) {
	items := []ModelItem{
		{ID: "gpt-4o-2024-08-06", CreatedAt: time.Unix(1715000000, 0).UTC(), OwnedBy: "openai"},
	}
	openaiBody, err := FormatModelList(OpenAI, items)
	if err != nil {
		t.Fatalf("FormatModelList(OpenAI) error: %v", err)
	}
	for _, tc := range []struct {
		name string
		p    Parser
	}{
		{"未实现 ModelsFormatter 的 Parser", nonFormatterParser{}},
		{"nil Parser", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, err := FormatModelList(tc.p, items)
			if err != nil {
				t.Fatalf("FormatModelList error: %v", err)
			}
			if !bytes.Equal(body, openaiBody) {
				t.Fatalf("回退结果 = %s, 期望 OpenAI 格式 %s", body, openaiBody)
			}
			assertOpenAIListShape(t, body, items)
		})
	}
}

// assertOpenAIListShape 校验 OpenAI 系（openai / responses）模型列表的字段名与取值。
func assertOpenAIListShape(t *testing.T, body []byte, items []ModelItem) {
	t.Helper()
	var top map[string]json.RawMessage
	if err := json.Unmarshal(body, &top); err != nil {
		t.Fatalf("解析模型列表响应失败: %v", err)
	}
	if len(top) != 2 || top["object"] == nil || top["data"] == nil {
		t.Fatalf("顶层字段 = %v, 期望恰好是 object 与 data", keysOf(top))
	}
	if got := string(top["object"]); got != `"list"` {
		t.Fatalf("object = %s, 期望 \"list\"", got)
	}
	var data []map[string]json.RawMessage
	if err := json.Unmarshal(top["data"], &data); err != nil {
		t.Fatalf("data 不是对象数组: %v", err)
	}
	if len(data) != len(items) {
		t.Fatalf("data 长度 = %d, 期望 %d", len(data), len(items))
	}
	for i, item := range items {
		entry := data[i]
		if len(entry) != 4 || entry["id"] == nil || entry["object"] == nil || entry["created"] == nil || entry["owned_by"] == nil {
			t.Fatalf("data[%d] 字段 = %v, 期望恰好是 id / object / created / owned_by", i, keysOf(entry))
		}
		entryBody, err := json.Marshal(entry)
		if err != nil {
			t.Fatalf("重新序列化 data[%d] 失败: %v", i, err)
		}
		var got struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		}
		if err := json.Unmarshal(entryBody, &got); err != nil {
			t.Fatalf("data[%d] 结构不符: %v", i, err)
		}
		if got.ID != item.ID || got.Object != "model" || got.OwnedBy != item.OwnedBy || got.Created != item.CreatedAt.Unix() {
			t.Fatalf("data[%d] = %+v, 期望 id=%s object=model owned_by=%s created=%d",
				i, got, item.ID, item.OwnedBy, item.CreatedAt.Unix())
		}
	}
}
