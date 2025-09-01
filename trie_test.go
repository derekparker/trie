package trie

import (
	"bufio"
	"log"
	"os"
	"sort"
	"testing"
)

func BenchmarkMaskruneslice(b *testing.B) {
	testCases := []struct {
		name  string
		input string
	}{
		{"short", "test"},
		{"medium", "benchmark"},
		{"long", "thisisaverylongstringfortesting"},
		{"alphabet", "abcdefghijklmnopqrstuvwxyz"},
	}

	for _, tc := range testCases {
		runes := []rune(tc.input)
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = maskruneslice(runes)
			}
		})
	}
}

func createTrieAndAddFromFile[T any](path string, val T) *Trie[T] {
	t := New[T]()
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	reader := bufio.NewScanner(file)

	for reader.Scan() {
		t.Add(reader.Text(), val)
	}

	if reader.Err() != nil {
		log.Fatal(err)
	}
	return t
}

func TestTrieAll(t *testing.T) {
	trie := New[int]()

	trie.Add("foo", 1)
	trie.Add("bar", 2)
	trie.Add("baz", 3)
	trie.Add("bur", 4)

	for key, value := range trie.AllKeyValuesIter() {
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

func TestAllKeyValues(t *testing.T) {
	trie := New[int]()

	trie.Add("foo", 1)
	trie.Add("bar", 2)
	trie.Add("baz", 3)
	trie.Add("bur", 4)

	kvMap := trie.AllKeyValues()

	// Check that we got all 4 entries
	if len(kvMap) != 4 {
		t.Errorf("Expected 4 entries, got: %d", len(kvMap))
	}

	// Check each key-value pair
	expectedPairs := map[string]int{
		"foo": 1,
		"bar": 2,
		"baz": 3,
		"bur": 4,
	}

	for key, expectedValue := range expectedPairs {
		if value, ok := kvMap[key]; !ok {
			t.Errorf("Key %s not found in result", key)
		} else if value != expectedValue {
			t.Errorf("For key %s: expected %d, got %d", key, expectedValue, value)
		}
	}

	// Check for unexpected keys
	for key := range kvMap {
		if _, ok := expectedPairs[key]; !ok {
			t.Errorf("Unexpected key in result: %s", key)
		}
	}
}

func TestAllKeyValuesEmpty(t *testing.T) {
	trie := New[int]()

	kvMap := trie.AllKeyValues()

	// Check that we got an empty map
	if len(kvMap) != 0 {
		t.Errorf("Expected empty map, got %d entries", len(kvMap))
	}
}

func TestAllKeyValuesWithDifferentValues(t *testing.T) {
	trie := New[string]()

	// Add keys with specific string values to verify correct mapping
	trie.Add("apple", "fruit")
	trie.Add("application", "software")
	trie.Add("apply", "verb")
	trie.Add("banana", "yellow fruit")
	trie.Add("bandana", "cloth")
	trie.Add("band", "music group")

	kvMap := trie.AllKeyValues()

	// Check that we got all 6 entries
	if len(kvMap) != 6 {
		t.Errorf("Expected 6 entries, got: %d", len(kvMap))
	}

	// Check each key-value pair for exact match
	expectedPairs := map[string]string{
		"apple":       "fruit",
		"application": "software",
		"apply":       "verb",
		"banana":      "yellow fruit",
		"bandana":     "cloth",
		"band":        "music group",
	}

	for key, expectedValue := range expectedPairs {
		if value, ok := kvMap[key]; !ok {
			t.Errorf("Key %s not found in result", key)
		} else if value != expectedValue {
			t.Errorf("For key %s: expected '%s', got '%s'", key, expectedValue, value)
		}
	}

	// Check for unexpected keys
	for key := range kvMap {
		if _, ok := expectedPairs[key]; !ok {
			t.Errorf("Unexpected key in result: %s", key)
		}
	}
}

func TestAllKeyValuesAndIteratorConsistency(t *testing.T) {
	trie := New[int]()

	// Add various keys with values
	testData := map[string]int{
		"apple":       1,
		"application": 2,
		"apply":       3,
		"banana":      4,
		"band":        5,
		"bandana":     6,
		"foo":         7,
		"foobar":      8,
		"foobaz":      9,
		"bar":         10,
	}

	for key, value := range testData {
		trie.Add(key, value)
	}

	// Get results from AllKeyValues
	mapResult := trie.AllKeyValues()

	// Collect results from AllKeyValuesIter
	iterResult := make(map[string]int)
	for key, value := range trie.AllKeyValuesIter() {
		iterResult[key] = value
	}

	// Check that both have the same number of entries
	if len(mapResult) != len(iterResult) {
		t.Errorf("Length mismatch: AllKeyValues returned %d entries, AllKeyValuesIter returned %d entries",
			len(mapResult), len(iterResult))
	}

	// Check that all entries match
	for key, mapValue := range mapResult {
		if iterValue, ok := iterResult[key]; !ok {
			t.Errorf("Key %s found in AllKeyValues but not in AllKeyValuesIter", key)
		} else if mapValue != iterValue {
			t.Errorf("Value mismatch for key %s: AllKeyValues=%d, AllKeyValuesIter=%d",
				key, mapValue, iterValue)
		}
	}

	// Check the reverse - all iterator entries are in the map
	for key := range iterResult {
		if _, ok := mapResult[key]; !ok {
			t.Errorf("Key %s found in AllKeyValuesIter but not in AllKeyValues", key)
		}
	}
}

func TestTrieAdd(t *testing.T) {
	trie := New[int]()

	n := trie.Add("foo", 1)

	if n.meta != 1 {
		t.Errorf("Expected 1, got: %d", n.meta)
	}
}

func TestTrieFind(t *testing.T) {
	trie := New[int]()
	trie.Add("foo", 1)

	n, ok := trie.Find("foo")
	if ok != true {
		t.Fatal("Could not find node")
	}

	if n.Val() != 1 {
		t.Errorf("Expected 1, got: %d", n.meta)
	}
}

func TestTrieFindMissingWithSubtree(t *testing.T) {
	trie := New[int]()
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
	trie := New[int]()
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
	trie := New[int]()

	n, ok := trie.Find("foo")
	if ok != false {
		t.Errorf("Expected ok to be false")
	}
	if n != nil {
		t.Errorf("Expected nil, got: %v", n)
	}
}

func TestRemove(t *testing.T) {
	trie := New[int]()
	initial := []string{"football", "foostar", "foosball"}

	for _, key := range initial {
		trie.Add(key, 0)
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

func TestRemoveRoot(t *testing.T) {
	trie := New[interface{}]()
	trie.Add("root", nil)
	trie.Remove("root")
	var ok bool
	_, ok = trie.Find("root")
	if ok {
		t.Error("Expected 0 keys")
	}

	// Try to write some data after the trie was purged
	trie.Add("root", nil)
	_, ok = trie.Find("root")
	if !ok {
		t.Error("Expected 1 keys")
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
			trie := New[interface{}]()
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
	trie := New[interface{}]()
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
	trie := New[interface{}]()
	keys := trie.PrefixSearch("")
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys from empty trie, got: %d", len(keys))
	}
}

func TestPrefixSearchIter(t *testing.T) {
	trie := New[string]()

	// Add test data with values
	testData := map[string]string{
		"foo":                          "value1",
		"foosball":                     "value2",
		"football":                     "value3",
		"foreboding":                   "value4",
		"forementioned":                "value5",
		"foretold":                     "value6",
		"foreverandeverandeverandever": "value7",
		"forbidden":                    "value8",
		"bar":                          "value9",
		"baz":                          "value10",
	}

	for key, value := range testData {
		trie.Add(key, value)
	}

	tests := []struct {
		prefix   string
		expected map[string]string
	}{
		{
			prefix: "fo",
			expected: map[string]string{
				"foo":                          "value1",
				"foosball":                     "value2",
				"football":                     "value3",
				"foreboding":                   "value4",
				"forementioned":                "value5",
				"foretold":                     "value6",
				"foreverandeverandeverandever": "value7",
				"forbidden":                    "value8",
			},
		},
		{
			prefix: "foosbal",
			expected: map[string]string{
				"foosball": "value2",
			},
		},
		{
			prefix: "bar",
			expected: map[string]string{
				"bar": "value9",
			},
		},
		{
			prefix:   "xyz",
			expected: map[string]string{},
		},
		{
			prefix:   "",
			expected: testData, // Empty prefix should return all entries
		},
	}

	for _, test := range tests {
		t.Run(test.prefix, func(t *testing.T) {
			// Collect results from iterator
			iterResults := make(map[string]string)
			for key, value := range trie.PrefixSearchIter(test.prefix) {
				iterResults[key] = value
			}

			// Compare lengths
			if len(iterResults) != len(test.expected) {
				t.Errorf("Length mismatch for prefix '%s': got %d, expected %d",
					test.prefix, len(iterResults), len(test.expected))
			}

			// Compare key-value pairs
			for expectedKey, expectedValue := range test.expected {
				if actualValue, ok := iterResults[expectedKey]; !ok {
					t.Errorf("Missing key '%s' for prefix '%s'", expectedKey, test.prefix)
				} else if actualValue != expectedValue {
					t.Errorf("Value mismatch for key '%s' with prefix '%s': got '%s', expected '%s'",
						expectedKey, test.prefix, actualValue, expectedValue)
				}
			}

			// Check for unexpected keys
			for actualKey := range iterResults {
				if _, ok := test.expected[actualKey]; !ok {
					t.Errorf("Unexpected key '%s' for prefix '%s'", actualKey, test.prefix)
				}
			}
		})
	}
}

func TestPrefixSearchIterEmpty(t *testing.T) {
	trie := New[string]()

	count := 0
	for range trie.PrefixSearchIter("") {
		count++
	}

	if count != 0 {
		t.Errorf("Expected 0 entries from empty trie, got: %d", count)
	}
}

func TestPrefixSearchIterEarlyStop(t *testing.T) {
	trie := New[int]()
	keys := []string{"foo", "foobar", "foobaz", "football", "foosball"}
	for i, key := range keys {
		trie.Add(key, i)
	}

	// Test that we can stop iteration early
	count := 0
	maxCount := 2
	for range trie.PrefixSearchIter("foo") {
		count++
		if count >= maxCount {
			break
		}
	}

	if count != maxCount {
		t.Errorf("Expected to stop at %d iterations, got %d", maxCount, count)
	}
}

func TestPrefixSearchAndIterConsistency(t *testing.T) {
	trie := New[int]()

	// Add test data
	testData := map[string]int{
		"apple":       1,
		"application": 2,
		"apply":       3,
		"banana":      4,
		"band":        5,
		"bandana":     6,
		"can":         7,
		"candy":       8,
		"candid":      9,
	}

	for key, value := range testData {
		trie.Add(key, value)
	}

	prefixes := []string{"", "app", "ban", "can", "z"}

	for _, prefix := range prefixes {
		t.Run(prefix, func(t *testing.T) {
			// Get results from PrefixSearch
			searchResults := trie.PrefixSearch(prefix)
			searchSet := make(map[string]bool)
			for _, key := range searchResults {
				searchSet[key] = true
			}

			// Collect results from PrefixSearchIter
			iterResults := make(map[string]int)
			for key, value := range trie.PrefixSearchIter(prefix) {
				iterResults[key] = value
			}

			// Check that all keys match
			if len(searchResults) != len(iterResults) {
				t.Errorf("Length mismatch for prefix '%s': PrefixSearch=%d, PrefixSearchIter=%d",
					prefix, len(searchResults), len(iterResults))
			}

			// Verify all keys from PrefixSearch are in PrefixSearchIter
			for _, key := range searchResults {
				if _, ok := iterResults[key]; !ok {
					t.Errorf("Key '%s' found in PrefixSearch but not in PrefixSearchIter for prefix '%s'",
						key, prefix)
				}
			}

			// Verify all keys from PrefixSearchIter are in PrefixSearch
			for key := range iterResults {
				if !searchSet[key] {
					t.Errorf("Key '%s' found in PrefixSearchIter but not in PrefixSearch for prefix '%s'",
						key, prefix)
				}
			}
		})
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

	trie := New[interface{}]()
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

func TestFuzzySearchIter(t *testing.T) {
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

	trie := New[interface{}]()
	for _, key := range setup {
		trie.Add(key, nil)
	}

	for _, test := range tests {
		t.Run(test.partial, func(t *testing.T) {
			// Collect results from iterator
			var results []string
			for key := range trie.FuzzySearchIter(test.partial) {
				results = append(results, key)
			}

			// Get results from regular FuzzySearch
			expected := trie.FuzzySearch(test.partial)

			// Check lengths match
			if len(results) != test.length {
				t.Errorf("Expected len(results) to == %d, was %d for %s results was %#v",
					test.length, len(results), test.partial, results)
			}

			// Check that results contain the same keys (order may differ)
			if len(results) != len(expected) {
				t.Errorf("Iterator results length %d doesn't match FuzzySearch length %d for %s",
					len(results), len(expected), test.partial)
			}

			// Create maps to check set equality
			resultSet := make(map[string]bool)
			for _, key := range results {
				resultSet[key] = true
			}

			expectedSet := make(map[string]bool)
			for _, key := range expected {
				expectedSet[key] = true
			}

			// Check that all expected keys are in results
			for key := range expectedSet {
				if !resultSet[key] {
					t.Errorf("Expected key %s not found in iterator results for pattern %s", key, test.partial)
				}
			}

			// Check that no unexpected keys are in results
			for key := range resultSet {
				if !expectedSet[key] {
					t.Errorf("Unexpected key %s found in iterator results for pattern %s", key, test.partial)
				}
			}
		})
	}
}

func TestFuzzySearchIterEarlyStop(t *testing.T) {
	trie := New[interface{}]()
	keys := []string{"foo", "foobar", "foobaz", "football", "foosball"}
	for _, key := range keys {
		trie.Add(key, nil)
	}

	// Test that we can stop iteration early
	count := 0
	maxCount := 2
	for range trie.FuzzySearchIter("f") {
		count++
		if count >= maxCount {
			break
		}
	}

	if count != maxCount {
		t.Errorf("Expected to stop at %d iterations, got %d", maxCount, count)
	}
}

func TestFuzzySearchEmpty(t *testing.T) {
	trie := New[interface{}]()
	keys := trie.FuzzySearch("")
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys from empty trie, got: %d", len(keys))
	}
}

func TestFuzzySearchSorting(t *testing.T) {
	trie := New[interface{}]()
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
	trie := New[interface{}]()
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
	trie := createTrieAndAddFromFile[interface{}]("/usr/share/dict/words", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = trie.PrefixSearch("fo")
	}
}

func BenchmarkPrefixSearchIter(b *testing.B) {
	trie := createTrieAndAddFromFile[interface{}]("/usr/share/dict/words", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		count := 0
		for range trie.PrefixSearchIter("fo") {
			count++
		}
	}
}

func BenchmarkPrefixSearchIterEarlyStop(b *testing.B) {
	trie := createTrieAndAddFromFile[interface{}]("/usr/share/dict/words", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		count := 0
		maxCount := 10
		for range trie.PrefixSearchIter("fo") {
			count++
			if count >= maxCount {
				break
			}
		}
	}
}

func BenchmarkFuzzySearch(b *testing.B) {
	trie := createTrieAndAddFromFile[interface{}]("fixtures/test.txt", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = trie.FuzzySearch("fs")
	}
}

func BenchmarkFuzzySearchIter(b *testing.B) {
	trie := createTrieAndAddFromFile[interface{}]("fixtures/test.txt", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		count := 0
		for range trie.FuzzySearchIter("fs") {
			count++
		}
	}
}

func BenchmarkFuzzySearchIterEarlyStop(b *testing.B) {
	trie := createTrieAndAddFromFile[interface{}]("fixtures/test.txt", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		count := 0
		maxCount := 10
		for range trie.FuzzySearchIter("fs") {
			count++
			if count >= maxCount {
				break
			}
		}
	}
}

func BenchmarkBuildTree(b *testing.B) {
	for i := 0; i < b.N; i++ {
		createTrieAndAddFromFile[interface{}]("/usr/share/dict/words", nil)
	}
}

func BenchmarkAllKeyValues(b *testing.B) {
	trie := createTrieAndAddFromFile[interface{}]("fixtures/test.txt", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = trie.AllKeyValues()
	}
}

func BenchmarkAllKeyValuesIter(b *testing.B) {
	trie := createTrieAndAddFromFile[interface{}]("fixtures/test.txt", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		count := 0
		for range trie.AllKeyValuesIter() {
			count++
		}
	}
}

func TestSupportChinese(t *testing.T) {
	trie := New[interface{}]()
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
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	scanner := bufio.NewScanner(f)
	var words []string
	for scanner.Scan() {
		word := scanner.Text()
		words = append(words, word)
	}
	b.ResetTimer()
	trie := New[interface{}]()
	for i := 0; i < b.N; i++ {
		trie.Add(words[i%len(words)], nil)
	}
}
