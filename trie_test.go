package trie

import (
	"bufio"
	"log"
	"os"
	"sort"
	"testing"
)

func addFromFile(t *Trie, path string) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	reader := bufio.NewScanner(file)

	for reader.Scan() {
		t.Add(reader.Text(), nil)
	}

	if reader.Err() != nil {
		log.Fatal(err)
	}
}

func TestTrieAdd(t *testing.T) {
	trie := New()

	n := trie.Add("foo", 1)

	if n.Meta().(int) != 1 {
		t.Errorf("Expected 1, got: %d", n.Meta().(int))
	}
}

func TestTrieFind(t *testing.T) {
	trie := New()
	trie.Add("foo", 1)

	n, ok := trie.Find("foo")
	if ok != true {
		t.Fatal("Could not find node")
	}

	if n.Meta().(int) != 1 {
		t.Errorf("Expected 1, got: %d", n.Meta().(int))
	}
}

func TestTrieFindMissingWithSubtree(t *testing.T) {
	trie := New()
	trie.Add("fooish", 1)
	trie.Add("foobar", 1)

	n, ok := trie.Find("foo")
	if ok != false {
		t.Errorf("Expected ok to be false")
	}
	if n != nil {
		t.Errorf("Expected nil, got: %v", n)
	}
}

func TestTrieHasKeysWithPrefix(t *testing.T) {
	trie := New()
	trie.Add("fooish", 1)
	trie.Add("foobar", 1)

	testcases := []struct {
		key      string
		expected bool
	}{
		{"foobar", true},
		{"foo", true},
		{"fool", false},
	}
	for _, testcase := range testcases {
		if trie.HasKeysWithPrefix(testcase.key) != testcase.expected {
			t.Errorf("HasKeysWithPrefix(\"%s\"): expected result to be %t", testcase.key, testcase.expected)
		}
	}
}

func TestTrieFindMissing(t *testing.T) {
	trie := New()

	n, ok := trie.Find("foo")
	if ok != false {
		t.Errorf("Expected ok to be false")
	}
	if n != nil {
		t.Errorf("Expected nil, got: %v", n)
	}
}

func TestRemove(t *testing.T) {
	trie := New()
	initial := []string{"football", "foostar", "foosball"}

	for _, key := range initial {
		trie.Add(key, nil)
	}

	trie.Remove("foosball")
	keys := trie.Keys()

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys got %d", len(keys))
	}

	for _, k := range keys {
		if k != "football" && k != "foostar" {
			t.Errorf("key was: %s", k)
		}
	}

	keys = trie.FuzzySearch("foo")
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys got %d", len(keys))
	}

	for _, k := range keys {
		if k != "football" && k != "foostar" {
			t.Errorf("Expected football got: %#v", k)
		}
	}
}

func TestTrieKeys(t *testing.T) {
	tableTests := []struct {
		name         string
		expectedKeys []string
	}{
		{"Two", []string{"bar", "foo"}},
		{"One", []string{"foo"}},
		{"Empty", []string{}},
	}

	for _, test := range tableTests {
		t.Run(test.name, func(t *testing.T) {
			trie := New()
			for _, key := range test.expectedKeys {
				trie.Add(key, nil)
			}

			keys := trie.Keys()
			if len(keys) != len(test.expectedKeys) {
				t.Errorf("Expected %v keys, got %d, keys were: %v", len(test.expectedKeys), len(keys), trie.Keys())
			}

			sort.Strings(keys)
			for i, key := range keys {
				if key != test.expectedKeys[i] {
					t.Errorf("Expected %#v, got %#v", test.expectedKeys[i], key)
				}
			}
		})
	}
}

func TestPrefixSearch(t *testing.T) {
	trie := New()
	expected := []string{
		"foo",
		"foosball",
		"football",
		"foreboding",
		"forementioned",
		"foretold",
		"foreverandeverandeverandever",
		"forbidden",
	}

	defer func() {
		r := recover()
		if r != nil {
			t.Error(r)
		}
	}()

	trie.Add("bar", nil)
	for _, key := range expected {
		trie.Add(key, nil)
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
		sort.Strings(actual)
		sort.Strings(test.expected)
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

func TestPrefixSearchEmpty(t *testing.T) {
	trie := New()
	keys := trie.PrefixSearch("")
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys from empty trie, got: %d", len(keys))
	}
}

func TestFuzzySearch(t *testing.T) {
	setup := []string{
		"foosball",
		"football",
		"bmerica",
		"ked",
		"kedlock",
		"frosty",
		"bfrza",
		"foo/bart/baz.go",
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
		{"ft", 3},
		{"fy", 1},
		{"fz", 2},
		{"a", 5},
		{"", 8},
		{"zzz", 0},
	}

	trie := New()
	for _, key := range setup {
		trie.Add(key, nil)
	}

	for _, test := range tests {
		t.Run(test.partial, func(t *testing.T) {
			actual := trie.FuzzySearch(test.partial)
			if len(actual) != test.length {
				t.Errorf("Expected len(actual) to == %d, was %d for %s actual was %#v",
					test.length, len(actual), test.partial, actual)
			}
		})
	}
}

func TestFuzzySearchEmpty(t *testing.T) {
	trie := New()
	keys := trie.FuzzySearch("")
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys from empty trie, got: %d", len(keys))
	}
}

func TestFuzzySearchSorting(t *testing.T) {
	trie := New()
	setup := []string{
		"foosball",
		"football",
		"bmerica",
		"ked",
		"kedlock",
		"frosty",
		"bfrza",
		"foo/bart/baz.go",
	}

	for _, key := range setup {
		trie.Add(key, nil)
	}

	actual := trie.FuzzySearch("fz")
	expected := []string{"bfrza", "foo/bart/baz.go"}

	if len(actual) != len(expected) {
		t.Fatalf("expected len %d got %d", len(expected), len(actual))
	}
	for i, v := range expected {
		if actual[i] != v {
			t.Errorf("Expected %s got %s", v, actual[i])
		}
	}

}

func BenchmarkTieKeys(b *testing.B) {
	trie := New()
	keys := []string{"bar", "foo", "baz", "bur", "zum", "burzum", "bark", "barcelona", "football", "foosball", "footlocker"}

	for _, key := range keys {
		trie.Add(key, nil)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.Keys()
	}
}

func BenchmarkPrefixSearch(b *testing.B) {
	trie := New()
	addFromFile(trie, "/usr/share/dict/words")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = trie.PrefixSearch("fo")
	}
}

func BenchmarkFuzzySearch(b *testing.B) {
	trie := New()
	addFromFile(trie, "/usr/share/dict/words")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = trie.FuzzySearch("fs")
	}
}

func BenchmarkBuildTree(b *testing.B) {
	for i := 0; i < b.N; i++ {
		trie := New()
		addFromFile(trie, "/usr/share/dict/words")
	}
}

func TestSupportChinese(t *testing.T) {
	trie := New()
	expected := []string{"苹果 沂水县", "苹果", "大蒜", "大豆"}

	for _, key := range expected {
		trie.Add(key, nil)
	}

	tests := []struct {
		pre      string
		expected []string
		length   int
	}{
		{"苹", expected[:2], len(expected[:2])},
		{"大", expected[2:], len(expected[2:])},
		{"大蒜", []string{"大蒜"}, 1},
	}

	for _, test := range tests {
		actual := trie.PrefixSearch(test.pre)
		sort.Strings(actual)
		sort.Strings(test.expected)
		if len(actual) != test.length {
			t.Errorf("Expected len(actual) to == %d for pre %s", test.length, test.pre)
		}

		for i, key := range actual {
			if key != test.expected[i] {
				t.Errorf("Expected %v got: %v", test.expected[i], key)
			}
		}
	}
}

func BenchmarkAdd(b *testing.B) {
	f, err := os.Open("/usr/share/dict/words")
	if err != nil {
		b.Fatal("couldn't open bag of words")
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var words []string
	for scanner.Scan() {
		word := scanner.Text()
		words = append(words, word)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie := New()
		for k := range words {
			trie.Add(words[k], nil)
		}
	}
}

func BenchmarkAddRemove(b *testing.B) {
	words := []string{"AAAA1", "AAAA2", "ABAA1", "AABA1", "ABAA2"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie := New()
		for k := range words {
			trie.Add(words[k], nil)
		}
		for k := range words {
			trie.Remove(words[k])
		}
	}
}
