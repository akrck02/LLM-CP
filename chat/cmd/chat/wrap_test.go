package main

import (
	"slices"
	"testing"
)

func TestAWordLongerThanALineIsCutIntoPiecesThatFit(test *testing.T) {
	cases := []struct {
		word string
		most int
		want []string
	}{
		{"docs", 10, []string{"docs"}},
		{"/home/alice/api", 5, []string{"/home", "/alic", "e/api"}},
		{"sesión", 3, []string{"ses", "ión"}},
	}
	for _, each := range cases {
		if got := wordPieces(each.word, each.most); !slices.Equal(got, each.want) {
			test.Errorf("wordPieces(%q, %d) = %q, want %q", each.word, each.most, got, each.want)
		}
	}
}
