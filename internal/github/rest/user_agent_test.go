package rest

import "testing"

func TestParseGHCLIVersion(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name: "standard gh version output",
			output: "gh version 2.86.0 (2026-01-21)\n" +
				"https://github.com/cli/cli/releases/tag/v2.86.0\n",
			want: "2.86.0",
		},
		{
			name:   "empty output",
			output: "",
			want:   "",
		},
		{
			name:   "non matching output",
			output: "some unexpected output",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseGHCLIVersion(tt.output); got != tt.want {
				t.Fatalf("parseGHCLIVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}
