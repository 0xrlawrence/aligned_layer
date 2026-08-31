package operator

import "testing"

func TestParseBatchURLs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty string yields no URLs",
			input: "",
			want:  []string{},
		},
		{
			name:  "single URL",
			input: "https://a.example/batch",
			want:  []string{"https://a.example/batch"},
		},
		{
			name:  "surrounding whitespace is trimmed and blanks skipped",
			input: " https://a.example/batch ,, https://b.example/batch ",
			want:  []string{"https://a.example/batch", "https://b.example/batch"},
		},
		{
			name:  "exactly MaxBatchURLs are kept",
			input: "u1,u2,u3,u4,u5",
			want:  []string{"u1", "u2", "u3", "u4", "u5"},
		},
		{
			name:  "more than MaxBatchURLs are truncated to the limit",
			input: "u1,u2,u3,u4,u5,u6,u7",
			want:  []string{"u1", "u2", "u3", "u4", "u5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseBatchURLs(tt.input)
			if len(got) > MaxBatchURLs {
				t.Fatalf("parseBatchURLs returned %d URLs, which exceeds MaxBatchURLs (%d)", len(got), MaxBatchURLs)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("parseBatchURLs(%q) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("parseBatchURLs(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}
