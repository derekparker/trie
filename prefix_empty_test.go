package trie_test

import (
	trie "github.com/derekparker/trie/v3"
	"testing"
)

func TestHasKeysWithEmptyPrefix(t *testing.T) {
	tr := trie.New[int]()
	check := func(want bool) {
		t.Helper()
		if got := tr.HasKeysWithPrefix(""); got != want {
			t.Errorf("HasKeysWithPrefix(\"\") = %t, want %t", got, want)
		}
		if got := len(tr.PrefixSearch("")) > 0; got != want {
			t.Errorf("PrefixSearch(\"\") has keys = %t, want %t", got, want)
		}
	}
	check(false)
	tr.Add("foobar", 1)
	tr.Add("foobaz", 2)
	check(true)
	tr.Remove("foobar")
	check(true)
	tr.Remove("foobaz")
	check(false)
	tr.Add("日本語", 3)
	check(true)
	tr.Remove("日本語")
	check(false)
}
