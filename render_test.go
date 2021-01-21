package web

import (
	"testing"
)

func TestExt2(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"filename.a", ""},
		{"filename.a.b", ".a"},
		{"filename.a.b.c", ".b"},
		{"/dir.a/file.b", ""},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			actual := ext2(tc.input)
			if actual != tc.expected {
				t.Errorf("actual=%s expected=%s", actual, tc.expected)
			}
		})
	}
}
