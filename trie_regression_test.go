package trie

import (
	"slices"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

// checkMaskInvariant verifies, for every node in the subtree, that the node's
// mask covers its own segment plus every descendant's segment. FuzzySearch
// prunes any subtree whose mask lacks a pattern rune, so a mask that
// under-reports its own segment silently drops valid matches.
func checkMaskInvariant[T any](t *testing.T, nd *node[T], path string) uint64 {
	t.Helper()

	want := maskruneslice([]rune(nd.segment))
	for _, c := range nd.children {
		want |= checkMaskInvariant(t, c, path+nd.segment)
	}
	if nd.mask&want != want {
		t.Errorf("node %q: mask %b is missing bits from its subtree, want superset of %b",
			path+nd.segment, nd.mask, want)
	}
	return want
}

// checkCounts verifies size matches the number of terminal nodes actually
// reachable in the tree.
func checkCounts[T any](t *testing.T, tr *Trie[T]) {
	t.Helper()

	var terminals int
	for range collectIter(tr.root) {
		terminals++
	}
	if tr.size != terminals {
		t.Errorf("size = %d, but %d terminal nodes are reachable", tr.size, terminals)
	}
}

// Splitting a node builds an intermediate that owns the common prefix. If that
// prefix is left out of the intermediate's mask, the fuzzy filter prunes the
// whole subtree.
func TestFuzzySearchAfterNodeSplit(t *testing.T) {
	trie := New[int]()
	trie.Add("abc", 1)
	trie.Add("abd", 2)

	checkMaskInvariant(t, trie.root, "")

	testcases := []struct {
		pattern  string
		expected []string
	}{
		{"ab", []string{"abc", "abd"}},
		{"a", []string{"abc", "abd"}},
		{"abc", []string{"abc"}},
		{"abd", []string{"abd"}},
		{"ac", []string{"abc"}},
		{"ad", []string{"abd"}},
		{"abe", nil},
	}
	for _, testcase := range testcases {
		got := trie.FuzzySearch(testcase.pattern)
		slices.Sort(got)
		if !slices.Equal(got, testcase.expected) {
			t.Errorf("FuzzySearch(%q) = %v, want %v", testcase.pattern, got, testcase.expected)
		}
	}
}

// Repeated splits along the same branch each introduce an intermediate node,
// so the mask has to stay correct as the chain deepens.
func TestFuzzySearchAfterRepeatedSplits(t *testing.T) {
	trie := New[int]()
	keys := []string{"abcdef", "abcdxy", "abcz", "abq", "ar"}
	for i, key := range keys {
		trie.Add(key, i)
	}

	checkMaskInvariant(t, trie.root, "")
	checkCounts(t, trie)

	for _, key := range keys {
		got := trie.FuzzySearch(key)
		if !slices.Contains(got, key) {
			t.Errorf("FuzzySearch(%q) = %v, want it to contain %q", key, got, key)
		}
	}

	got := trie.FuzzySearch("abc")
	slices.Sort(got)
	want := []string{"abcz", "abcdef", "abcdxy"}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("FuzzySearch(\"abc\") = %v, want %v", got, want)
	}
}

// The mask invariant must survive a realistic build, not just handmade cases.
func TestMaskInvariantOnFixture(t *testing.T) {
	trie := createTrieAndAddFromFile("fixtures/test.txt", 0)
	checkMaskInvariant(t, trie.root, "")
	checkCounts(t, trie)
}

// findNode used to return a node whenever the key ran out partway through a
// compressed segment, so Find reported a bare prefix of a sole key as present.
func TestTrieFindPrefixOfCompressedSegment(t *testing.T) {
	trie := New[int]()
	trie.Add("foobar", 1)

	for _, key := range []string{"f", "fo", "foo", "fooba"} {
		if n, ok := trie.Find(key); ok {
			t.Errorf("Find(%q) = (%v, true), want false: %q is only a prefix", key, n, key)
		}
	}

	if _, ok := trie.Find("foobar"); !ok {
		t.Error("Find(\"foobar\") = false, want true")
	}

	// Prefix queries must still match inside the segment.
	for _, key := range []string{"f", "fo", "foo", "fooba", "foobar"} {
		if !trie.HasKeysWithPrefix(key) {
			t.Errorf("HasKeysWithPrefix(%q) = false, want true", key)
		}
	}
	if trie.HasKeysWithPrefix("foobarx") {
		t.Error("HasKeysWithPrefix(\"foobarx\") = true, want false")
	}
}

// Same root cause: Remove of a bare prefix used to land on the compressed node
// and delete the real key stored there.
func TestRemovePrefixOfCompressedSegment(t *testing.T) {
	trie := New[int]()
	trie.Add("foobar", 1)

	trie.Remove("foo")

	if _, ok := trie.Find("foobar"); !ok {
		t.Error("Remove(\"foo\") deleted the unrelated key \"foobar\"")
	}
	if trie.size != 1 {
		t.Errorf("size = %d after no-op Remove, want 1", trie.size)
	}
	checkCounts(t, trie)

	trie.Remove("foobar")
	if _, ok := trie.Find("foobar"); ok {
		t.Error("Remove(\"foobar\") did not delete it")
	}
	checkCounts(t, trie)
}

// Add incremented size before knowing whether the key was already present.
func TestTrieAddDuplicateKey(t *testing.T) {
	trie := New[int]()
	trie.Add("foo", 1)
	trie.Add("foo", 2)

	if trie.size != 1 {
		t.Errorf("size = %d after adding \"foo\" twice, want 1", trie.size)
	}
	checkCounts(t, trie)

	n, ok := trie.Find("foo")
	if !ok {
		t.Fatal("Find(\"foo\") = false, want true")
	}
	if n.Val() != 2 {
		t.Errorf("Val() = %d, want 2 (the re-Added value)", n.Val())
	}
	if keys := trie.Keys(); len(keys) != 1 {
		t.Errorf("Keys() = %v, want exactly one key", keys)
	}
}

// Duplicate Adds land on every kind of terminal: a fresh child, a node reached
// by exact segment match, and a node created by a split.
func TestTrieAddDuplicatesAcrossTerminalKinds(t *testing.T) {
	keys := []string{"foo", "foobar", "foobaz", "fo", "f", "quux"}

	trie := New[int]()
	for i, key := range keys {
		trie.Add(key, i)
	}
	if trie.size != len(keys) {
		t.Errorf("size = %d, want %d", trie.size, len(keys))
	}
	checkCounts(t, trie)

	// Re-add every key; nothing should be counted twice.
	for i, key := range keys {
		trie.Add(key, i*100)
	}
	if trie.size != len(keys) {
		t.Errorf("size = %d after re-adding every key, want %d", trie.size, len(keys))
	}
	checkCounts(t, trie)
	checkMaskInvariant(t, trie.root, "")

	for i, key := range keys {
		n, ok := trie.Find(key)
		if !ok {
			t.Errorf("Find(%q) = false, want true", key)
			continue
		}
		if n.Val() != i*100 {
			t.Errorf("Find(%q).Val() = %d, want %d", key, n.Val(), i*100)
		}
	}

	got := trie.Keys()
	slices.Sort(got)
	want := slices.Clone(keys)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("Keys() = %v, want %v", got, want)
	}
}

// Hammer Add/Remove/duplicate-Add against a reference map, checking the size
// and the mask invariant after every mutation. This
// exercises the split, exact-match and fresh-child paths together.
func TestMixedMutationsKeepInvariants(t *testing.T) {
	trie := New[int]()
	want := map[string]int{}

	// A deterministic pseudo-random walk over a set of overlapping keys, so
	// splits and merges happen in varied orders.
	keys := []string{
		"a", "ab", "abc", "abcd", "abd", "ac", "b", "ba", "bab", "bad",
		"cat", "cart", "car", "care", "cared", "dog", "do", "door", "doom",
	}
	seed := uint32(1)
	next := func(n int) int {
		seed = seed*1664525 + 1013904223
		return int(seed>>16) % n
	}

	for step := range 3000 {
		key := keys[next(len(keys))]
		if next(3) == 0 {
			trie.Remove(key)
			delete(want, key)
		} else {
			trie.Add(key, step)
			want[key] = step
		}

		if trie.size != len(want) {
			t.Fatalf("step %d: size = %d, want %d", step, trie.size, len(want))
		}
	}

	checkCounts(t, trie)
	checkMaskInvariant(t, trie.root, "")

	// Every surviving key must be findable with its latest value, and every
	// removed key must be gone.
	for _, key := range keys {
		n, ok := trie.Find(key)
		expected, shouldExist := want[key]
		if ok != shouldExist {
			t.Errorf("Find(%q) = %t, want %t", key, ok, shouldExist)
			continue
		}
		if ok && n.Val() != expected {
			t.Errorf("Find(%q).Val() = %d, want %d", key, n.Val(), expected)
		}
	}

	got := trie.Keys()
	slices.Sort(got)
	wantKeys := make([]string, 0, len(want))
	for key := range want {
		wantKeys = append(wantKeys, key)
	}
	slices.Sort(wantKeys)
	if !slices.Equal(got, wantKeys) {
		t.Errorf("Keys() = %v, want %v", got, wantKeys)
	}

	// Fuzzy search must still reach every surviving key.
	for _, key := range wantKeys {
		if !slices.Contains(trie.FuzzySearch(key), key) {
			t.Errorf("FuzzySearch(%q) does not contain %q", key, key)
		}
	}
}

// Segments are split at byte offsets, so a split must never land inside a
// multi-byte rune. These keys share the leading byte of their final rune but
// differ in its continuation byte, which is precisely the case that a naive
// byte-wise common-prefix length gets wrong.
func TestSplitInsideMultiByteRune(t *testing.T) {
	// "é" is 0xC3 0xA9 and "ê" is 0xC3 0xAA: same lead byte, different rune.
	// "日" is 0xE6 0x97 0xA5 and "旦" is 0xE6 0x97 0xA6: two shared bytes.
	testcases := [][]string{
		{"aé", "aê"},
		{"aé", "aêb"},
		{"x日", "x旦"},
		{"x日y", "x旦y"},
		{"日本語", "日本語です", "日本"},
		{"éa", "éb", "é"},
	}

	for _, keys := range testcases {
		trie := New[int]()
		for i, key := range keys {
			trie.Add(key, i)
		}

		checkCounts(t, trie)
		checkMaskInvariant(t, trie.root, "")
		checkSegmentsAreValidUTF8(t, trie.root)

		for i, key := range keys {
			n, ok := trie.Find(key)
			if !ok {
				t.Errorf("keys %q: Find(%q) = false, want true", keys, key)
				continue
			}
			if n.Val() != i {
				t.Errorf("keys %q: Find(%q).Val() = %d, want %d", keys, key, n.Val(), i)
			}
		}

		got := trie.Keys()
		slices.Sort(got)
		want := slices.Clone(keys)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Errorf("keys %q: Keys() = %q, want %q", keys, got, want)
		}
	}
}

// checkSegmentsAreValidUTF8 catches a split that cut a rune in half.
func checkSegmentsAreValidUTF8[T any](t *testing.T, nd *node[T]) {
	t.Helper()

	if !utf8.ValidString(nd.segment) {
		t.Errorf("node segment %q is not valid UTF-8; a split cut a rune in half", nd.segment)
	}
	for r, c := range nd.children {
		first, _ := utf8.DecodeRuneInString(c.segment)
		if first != r {
			t.Errorf("child keyed by %q but its segment %q starts with %q", r, c.segment, first)
		}
		checkSegmentsAreValidUTF8(t, c)
	}
}

// Reassembling a key from the segments along its path must reproduce it
// exactly, which fails if any segment boundary is misplaced.
func TestSegmentsReassembleIntoKeys(t *testing.T) {
	keys := []string{
		"日本語", "日本", "にほんご", "car", "cart", "carts", "cat",
		"", "a", "aé", "aê", "🙂", "🙃", "🙂🙃", "naïve", "naive",
	}

	trie := New[int]()
	added := map[string]bool{}
	for i, key := range keys {
		trie.Add(key, i)
		if key != "" {
			added[key] = true
		}
	}

	var walk func(nd *node[int], prefix string)
	walk = func(nd *node[int], prefix string) {
		path := prefix + nd.segment
		if nd.path != nil {
			if *nd.path != path {
				t.Errorf("node stores key %q but its path spells %q", *nd.path, path)
			}
			if !added[path] {
				t.Errorf("trie contains unexpected key %q", path)
			}
		}
		for _, c := range nd.children {
			walk(c, path)
		}
	}
	walk(trie.root, "")

	checkCounts(t, trie)
	checkSegmentsAreValidUTF8(t, trie.root)
}

// Keys() held the read lock and then called PrefixSearch, which took it again.
// sync.RWMutex read locks are not reentrant, so a writer arriving between the
// two acquisitions blocks both permanently.
func TestKeysConcurrentWithWriter(t *testing.T) {
	trie := New[int]()
	trie.Add("foo", 1)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 100000 {
			trie.Add("bar", 1)
			trie.Remove("bar")
		}
	}()
	go func() {
		for range 100000 {
			trie.Keys()
		}
	}()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("deadlock: Keys() recursively acquired the read lock while a writer was pending")
	}
}

// The iterator factories took the read lock and released it on return, leaving
// the actual iteration unsynchronized. Run this under -race.
func TestIteratorsHoldLockDuringIteration(t *testing.T) {
	trie := New[int]()
	trie.Add("foo", 1)
	trie.Add("foobar", 2)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := range 5000 {
			trie.Add("bar", i)
			trie.Remove("bar")
		}
	}()
	go func() {
		defer wg.Done()
		for range 5000 {
			for range trie.PrefixSearchIter("fo") {
			}
			for range trie.FuzzySearchIter("f") {
			}
			for range trie.AllKeyValuesIter() {
			}
		}
	}()

	wg.Wait()
}

// Breaking out of an iterator early must still release the read lock.
func TestIteratorsReleaseLockOnEarlyStop(t *testing.T) {
	trie := New[int]()
	for _, key := range []string{"foo", "foobar", "foobaz"} {
		trie.Add(key, 1)
	}

	for range trie.PrefixSearchIter("foo") {
		break
	}
	for range trie.FuzzySearchIter("f") {
		break
	}
	for range trie.AllKeyValuesIter() {
		break
	}

	// If any read lock leaked, this write blocks forever.
	done := make(chan struct{})
	go func() {
		defer close(done)
		trie.Add("quux", 1)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("an iterator leaked its read lock: a subsequent Add blocked")
	}
}
