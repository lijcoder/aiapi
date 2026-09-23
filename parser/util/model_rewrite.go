package util

import (
	"encoding/json"
	"errors"
)

// ErrModelNotString 表示请求体顶层 model 存在但不是 JSON 字符串，无法替换。
var ErrModelNotString = errors.New("model field is not a string")

// ReplaceTopLevelModel 返回把请求体顶层 model 替换为 name 的新 body。
//
// 以下情况直接原样返回入参 body（不改写、不重新编码，保证字节零变动）：
//   - body 为空、非法 JSON 或不是 JSON 对象
//   - 顶层没有 model 字段
//   - 顶层 model 已等于 name
//
// 需要改写时按 map[string]json.RawMessage 解码后重新编码：除 model 外的字段以
// 原始字节透传，数字精度与嵌套结构（messages、base64 图片等）不受影响；仅顶层
// 键序与 '<' '>' '&' 的转义形式可能与原 body 不同，JSON 语义等价，上游解析结果一致。
// 这样设计是为了让「未配置 provider_model」的绝大多数请求完全零开销。
func ReplaceTopLevelModel(body []byte, name string) ([]byte, error) {
	if len(body) == 0 {
		return body, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		// 非法 JSON / 非对象（数组、字符串、数字）：定位不到 model，交由上游处理
		return body, nil
	}
	raw, ok := fields["model"]
	if !ok {
		return body, nil
	}
	var current string
	if err := json.Unmarshal(raw, &current); err != nil {
		return nil, ErrModelNotString
	}
	if current == name {
		return body, nil
	}
	encoded, err := json.Marshal(name)
	if err != nil {
		return nil, err
	}
	fields["model"] = encoded
	rewritten, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	return rewritten, nil
}
