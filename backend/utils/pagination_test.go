package utils

import (
	"testing"

	"github.com/yourusername/kor-assetforge/constants"
)

func TestClampPageSize(t *testing.T) {
	cases := []struct {
		name  string
		input int
		want  int
	}{
		{"zero falls back to default", 0, constants.DefaultPageSize},
		{"negative falls back to default", -5, constants.DefaultPageSize},
		{"valid value is preserved", 42, 42},
		{"value at max is preserved", constants.MAX_PAGE_SIZE, constants.MAX_PAGE_SIZE},
		{"value over max is capped", constants.MAX_PAGE_SIZE + 1, constants.MAX_PAGE_SIZE},
		{"very large value is capped", 100000, constants.MAX_PAGE_SIZE},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClampPageSize(tc.input); got != tc.want {
				t.Errorf("ClampPageSize(%d) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}
