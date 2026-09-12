package parser

import "testing"

func TestParseStreamUsage_SkipsStreamWithoutUsage(t *testing.T) {
	usage, err := ParseStreamUsage(OpenAI, []byte("data: {\"id\":\"chunk_1\",\"model\":\"gpt-test\",\"choices\":[]}\n\ndata: [DONE]\n\n"))
	if err != nil {
		t.Fatalf("ParseStreamUsage error: %v", err)
	}
	if usage != nil {
		t.Fatalf("usage = %+v, want nil", usage)
	}
}

func TestParseStreamUsage_ResponsesCRLF(t *testing.T) {
	body := []byte("data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_1\",\"model\":\"gpt-4o\"}}\r\n\r\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"model\":\"gpt-4o\",\"usage\":{\"input_tokens\":7,\"output_tokens\":3,\"input_tokens_details\":{\"cached_tokens\":2}}}}\r\n\r\n")
	usage, err := ParseStreamUsage(Responses, body)
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
