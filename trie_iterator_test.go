//go:build goexperiment.rangefunc

package trie

import "testing"

func TestTrieAll(t *testing.T) {
	trie := New[int]()

	trie.Add("foo", 1)
	trie.Add("bar", 2)
	trie.Add("baz", 3)
	trie.Add("bur", 4)

	for key, value := range trie.All() {
		switch key {
		case "foo":
			if value != 1 {
				t.Errorf("Expected 1, got: %d", value)
			}
		case "bar":
			if value != 2 {
				t.Errorf("Expected 2, got: %d", value)
			}
		case "baz":
			if value != 3 {
				t.Errorf("Expected 3, got: %d", value)
			}
		case "bur":
			if value != 4 {
				t.Errorf("Expected 4, got: %d", value)
			}
		default:
			t.Errorf("Unexpected key: %s", key)
		}
	}
}
