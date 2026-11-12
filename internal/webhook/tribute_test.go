package webhook

import "testing"

func TestExtractPremiumCode(t *testing.T) {
	cases := []struct {
		name     string
		msg      string
		wantCode string
		wantRest string
	}{
		{
			name:     "code with nickname",
			msg:      "LZ-ABCD-2345 Vasya",
			wantCode: "LZ-ABCD-2345",
			wantRest: "Vasya",
		},
		{
			name:     "lowercase code in the middle",
			msg:      "ник: Вася, код lz-abcd-2345",
			wantCode: "LZ-ABCD-2345",
			wantRest: "ник: Вася, код",
		},
		{
			name:     "no code",
			msg:      "Вася",
			wantCode: "",
			wantRest: "Вася",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, rest := extractPremiumCode(tc.msg)
			if code != tc.wantCode {
				t.Fatalf("code = %q, want %q", code, tc.wantCode)
			}
			if rest != tc.wantRest {
				t.Fatalf("rest = %q, want %q", rest, tc.wantRest)
			}
		})
	}
}
