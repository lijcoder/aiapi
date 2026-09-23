package util

import "testing"

func TestSSEParseData(t *testing.T) {
	cases := []struct {
		name    string
		line    string
		want    string
		wantNil bool
	}{
		{"data with space", "data: {\"a\":1}", `{"a":1}`, false},
		{"data without space", "data:{\"a\":1}", `{"a":1}`, false},
		{"empty data", "data: ", "", false},
		{"event line", "event: message_start", "", true},
		{"id line", "id: 1", "", true},
		{"comment", ": ping", "", true},
		{"blank", "   ", "", true},
		{"plain text", "hello", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SSEParseData([]byte(tc.line))
			if tc.wantNil {
				if got != nil {
					t.Fatalf("SSEParseData(%q) = %q, want nil", tc.line, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("SSEParseData(%q) = nil, want %q", tc.line, tc.want)
			}
			if string(got) != tc.want {
				t.Fatalf("SSEParseData(%q) = %q, want %q", tc.line, got, tc.want)
			}
		})
	}
}

func TestEachSSEData(t *testing.T) {
	// 多行事件、event: 行、注释、空载荷与 CRLF 混合：只回调 data 载荷
	body := "event: message_start\r\n" +
		"data: {\"a\":1}\r\n" +
		"id: 1\r\n\r\n" +
		": ping\n\n" +
		"data: [DONE]\n\n" +
		"event: x\n" +
		"data: \n\n" +
		"data: {\"b\":2}\n\n"

	var got []string
	EachSSEData([]byte(body), func(data []byte) {
		got = append(got, string(data))
	})

	want := []string{`{"a":1}`, "[DONE]", `{"b":2}`}
	if len(got) != len(want) {
		t.Fatalf("callbacks = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("callback %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSplitSSEEvents_NormalizesLineEndings(t *testing.T) {
	cases := []struct {
		name string
		body string
		want []string
	}{
		{"LF", "data: a\n\ndata: b\n\n", []string{"data: a", "data: b"}},
		{"CRLF", "data: a\r\n\r\ndata: b\r\n\r\n", []string{"data: a", "data: b"}},
		{"CR", "data: a\r\rdata: b\r\r", []string{"data: a", "data: b"}},
		{"no trailing blank", "data: a\n\ndata: b", []string{"data: a", "data: b"}},
		{"empty body", "", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SplitSSEEvents([]byte(tc.body))
			if len(got) != len(tc.want) {
				t.Fatalf("SplitSSEEvents(%q) = %d events, want %d", tc.body, len(got), len(tc.want))
			}
			for i, want := range tc.want {
				if string(got[i]) != want {
					t.Fatalf("event %d = %q, want %q", i, got[i], want)
				}
			}
		})
	}
}
