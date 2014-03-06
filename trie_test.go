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

func TestKeysWithPrefix(t *testing.T) {
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

	actual := trie.KeysWithPrefix("fo")
	if len(actual) != len(expected) {
		t.Errorf("Expected len(actual) to == %d.", len(expected))
	}

	for i, key := range expected {
		if actual[i] != key {
			t.Errorf("Expected %v got: %v", key, actual[i])
		}
	}

	actual = trie.KeysWithPrefix("foosbal")
	if len(actual) != 1 {
		t.Errorf("Expected len(actual) to == %d.", 1)
	}

	for _, key := range actual {
		if key != "foosball" {
			t.Errorf("Expected %v got: %v", "foosball", key)
		}
	}

	actual = trie.KeysWithPrefix("footb")
	if len(actual) != 1 {
		t.Errorf("Expected len(actual) to == %d.", 1)
	}

	for _, key := range actual {
		if key != "football" {
			t.Errorf("Expected %v got: %v", "football", key)
		}
	}

	actual = trie.KeysWithPrefix("abc")
	if len(actual) != 0 {
		t.Errorf("Expected len(actual) to == %d.", 0)
	}

	trie.KeysWithPrefix("fsfsdfasdf")
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

func BenchmarkKeysWithPrefix(b *testing.B) {
	trie := CreateTrie()
	expected := []string{
		"foosball",
		"football",
		"foreboding",
		"forementioned",
		"foretold",
		"foreverandeverandeverandever",
		"forbidden",
		"forsupercalifragilisticexpyaladocious",
		"forsupercalifragilisticexpyaladocious",
		"forsupercalifragilisticexpyaladocious/fors",
		"forsupercalifragilisticexpyvlsdocious/fors",
		"fofsupercrlifralilisticexpyaladocgous",
		"foo/bar/baz/ber/her/mer/fur/a.out",
		"foo/baz/baz/ber/her/mer/fur/a.out",
		"foo/baz/bur/ber/her/mer/fur/a.out",
		"foo/baz/bur/sher/her/mer/fur/a.out",
		"foo/curr/bur/sher/her/mer/fur/a.out",
		"foo/lurr/bur/sher/her/mer/fur/a.out",
		"foo/turr/bur/sher/her/mer/fur/a.out",
		"foors",
	}

	trie.AddKey("bar")
	for _, key := range expected {
		trie.AddKey(key)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = trie.KeysWithPrefix("fo")
	}
}
