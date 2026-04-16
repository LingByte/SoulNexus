package llm

import "testing"

func TestExtractJSONFromLLMOutput(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`{"query":"x","filters":{}}`, `{"query":"x","filters":{}}`},
		{"prefix {\"query\":\"y\"} suffix", `{"query":"y"}`},
		{"```json\n{\"a\":1}\n```", `{"a":1}`},
	}
	for _, tc := range cases {
		got := ExtractJSONFromLLMOutput(tc.in, 0)
		if got != tc.want {
			t.Fatalf("in=%q want=%q got=%q", tc.in, tc.want, got)
		}
	}
}
