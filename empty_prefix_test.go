package trie

import "testing"

// HasKeysWithPrefix("") must only report true when the trie actually holds at
// least one key: findNode resolves the empty prefix to the root even when no
// terminal nodes exist, so without a size guard an empty trie (or a trie that
// has just been emptied) reports a prefix it does not have, disagreeing with
// PrefixSearch("").
func TestHasKeysWithPrefixEmptyPrefix(t *testing.T) {
	tr := New[int]()
	check := func(want bool) {
		t.Helper()
		if got := tr.HasKeysWithPrefix(""); got != want {
			t.Errorf("HasKeysWithPrefix(\"\") = %v, want %v", got, want)
		}
		if got := len(tr.PrefixSearch("")) > 0; got != want {
			t.Errorf("PrefixSearch(\"\") has keys = %v, want %v", got, want)
		}
	}

	check(false) // fresh trie

	tr.Add("foobar", 1)
	tr.Add("foobaz", 2)
	check(true)

	tr.Remove("foobar")
	check(true)

	tr.Remove("foobaz")
	check(false) // emptied trie

	tr.Add("日本語", 3)
	check(true)

	tr.Remove("日本語")
	check(false)
}