package trie

import (
	"testing"
)

func TestTrieAdd(t *testing.T) {
	trie := CreateTrie()

	i := trie.AddKey("foo")

	if i != 3 {
		t.Errorf("Expected 3, got: %d", i)
	}
}

func TestTrieAddFromFile(t *testing.T) {
	path := "fixtures/test.txt"
	expected := []string{"foo", "bar", "baz"}

	trie := CreateTrie()
	trie.AddKeysFromFile(path)
	keys := trie.Keys()

	kl := len(keys)
	if kl != 3 {
		t.Errorf("Expected 3 keys, got %d, keys were: %v", kl, trie.Keys())
	}

	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("Expected %#v, got %#v", expected[i], key)
		}
	}
}

func TestTrieKeys(t *testing.T) {
	trie := CreateTrie()
	expected := []string{"bar", "foo"}

	for _, key := range expected {
		trie.AddKey(key)
	}

	kl := len(trie.Keys())
	if kl != 2 {
		t.Errorf("Expected 2 keys, got %d, keys were: %v", kl, trie.Keys())
	}

	for i, key := range trie.Keys() {
		if key != expected[i] {
			t.Errorf("Expected %#v, got %#v", expected[i], key)
		}
	}
}

func TestPrefixSearch(t *testing.T) {
	trie := CreateTrie()
	expected := []string{"foosball", "football", "foreboding", "forementioned", "foretold", "foreverandeverandeverandever", "forbidden"}
	defer func() {
		r := recover()
		if r != nil {
			t.Error(r)
		}
	}()

	trie.AddKey("bar")
	for _, key := range expected {
		trie.AddKey(key)
	}

	tests := []struct {
		pre      string
		expected []string
		length   int
	}{
		{"fo", expected, len(expected)},
		{"foosbal", []string{"foosball"}, 1},
		{"abc", []string{}, 0},
	}

	for _, test := range tests {
		actual := trie.PrefixSearch(test.pre)
		if len(actual) != test.length {
			t.Errorf("Expected len(actual) to == %d for pre %s", test.length, test.pre)
		}

		for i, key := range actual {
			if key != test.expected[i] {
				t.Errorf("Expected %v got: %v", test.expected[i], key)
			}
		}
	}

	trie.PrefixSearch("fsfsdfasdf")
}

func TestFuzzySearch(t *testing.T) {
	trie := CreateTrie()
	setup := []string{
		"foosball",
		"football",
		"bmerica",
		"ked",
		"kedlock",
		"frosty",
		"bfrza",
		"foo/bar/baz.go",
	}
	tests := []struct {
		partial string
		length  int
	}{
		{"fsb", 1},
		{"footbal", 1},
		{"football", 1},
		{"fs", 2},
		{"oos", 1},
		{"kl", 1},
		{"ft", 2},
		{"fy", 1},
		{"fz", 2},
		{"a", 5},
	}

	for _, key := range setup {
		trie.AddKey(key)
	}

	for _, test := range tests {
		actual := trie.FuzzySearch(test.partial)

		if len(actual) != test.length {
			t.Errorf("Expected len(actual) to == %d, was %d for %s", test.length, len(actual), test.partial)
		}
	}
}

func BenchmarkTieKeys(b *testing.B) {
	trie := CreateTrie()
	keys := []string{"bar", "foo", "baz", "bur", "zum", "burzum", "bark", "barcelona", "football", "foosball", "footlocker"}

	for _, key := range keys {
		trie.AddKey(key)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.Keys()
	}
}

func BenchmarkPrefixSearch(b *testing.B) {
	trie := CreateTrie()
	trie.AddKeysFromFile("/usr/share/dict/words")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = trie.PrefixSearch("fo")
	}
}

func BenchmarkFuzzySearch(b *testing.B) {
	trie := CreateTrie()
	trie.AddKeysFromFile("/usr/share/dict/words")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = trie.FuzzySearch("fs")
	}
}
