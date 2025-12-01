package gemini

import (
	"encoding/json"
	"os"
	"testing"
)

func TestJsonParametersJsonSchema(t *testing.T) {
	paramsSchema := ParametersJsonSchema{
		Type: "object",
		Properties: map[string]any{
			"name": map[string]any{
				"type": "string",
			},
			"age": map[string]any{
				"type": "integer",
			},
		},
		Required: []string{"name"},
	}
	// JSON 序列化
	jsonData, err := json.Marshal(paramsSchema)
	if err != nil {
		panic(err)
	}
	t.Log("序列化结果：", string(jsonData))
}

func TestJsonRequestBody(t *testing.T) {
	// 读取 gemini_req.json 文件
	data, err := os.ReadFile("resources/gemini_req.json")
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}

	// 反序列化为 RequestBody 结构体
	var requestBody Request
	err = json.Unmarshal(data, &requestBody)
	if err != nil {
		t.Fatalf("JSON 反序列化失败: %v", err)
	}

	t.Logf("反序列化结果: %+v", requestBody)
}

func TestJsonSerializeDeserialize(t *testing.T) {
	// 读取 gemini_req.json 文件
	data, err := os.ReadFile("resources/gemini_req.json")
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}

	// 反序列化为 RequestBody 结构体
	var requestBody Request
	err = json.Unmarshal(data, &requestBody)
	if err != nil {
		t.Fatalf("JSON 反序列化失败: %v", err)
	}

	// 序列化回 JSON
	jsonData, err := json.MarshalIndent(requestBody, "", "    ")
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	// 写入到新文件
	err = os.WriteFile("gemini_req_de.json", jsonData, 0644)
	if err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}

	t.Log("序列化完成，已写入 gemini_req_de.json")
}

func TestJsonResponseDeserialize(t *testing.T) {
	// 读取 gemini_resp.json 文件
	data, err := os.ReadFile("resources/gemini_resp.json")
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}

	// 反序列化为 Response 结构体
	var response Response
	err = json.Unmarshal(data, &response)
	if err != nil {
		t.Fatalf("JSON 反序列化失败: %v", err)
	}

	t.Logf("反序列化结果: %+v", response)
}

func TestJsonResponseSerializeDeserialize(t *testing.T) {
	// 读取 gemini_resp.json 文件
	data, err := os.ReadFile("resources/gemini_resp.json")
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}

	// 反序列化为 Response 结构体
	var response Response
	err = json.Unmarshal(data, &response)
	if err != nil {
		t.Fatalf("JSON 反序列化失败: %v", err)
	}

	// 序列化回 JSON
	jsonData, err := json.MarshalIndent(response, "", "    ")
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	// 写入到新文件
	err = os.WriteFile("resources/gemini_resp_de.json", jsonData, 0644)
	if err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}

	t.Log("序列化完成，已写入 gemini_resp_de.json")
}
