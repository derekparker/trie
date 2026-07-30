// Implementation of an R-Way Trie data structure.
//
// A Trie has a root Node which is the base of the tree.
// Each subsequent Node has a letter and children, which are
// nodes that have letter values associated with them.
package trie

import (
	"iter"
	"maps"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"
)

type node[T any] struct {
	mask     uint64
	parent   *node[T]
	children map[rune]*node[T] // keyed by first rune of child's segment
	meta     T
	path     *string // pointer to full key for terminal nodes

	segment string // the string segment stored in this node
}

// Trie is a data structure that stores a set of strings.
type Trie[T any] struct {
	mu   sync.RWMutex
	root *node[T]
	size int
}

// ByKeys orders keys shortest first, breaking ties lexicographically so that
// results are stable rather than dependent on map iteration order.
type ByKeys []string

func (a ByKeys) Len() int      { return len(a) }
func (a ByKeys) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByKeys) Less(i, j int) bool {
	if len(a[i]) != len(a[j]) {
		return len(a[i]) < len(a[j])
	}
	return a[i] < a[j]
}

// New creates a new Trie with an initialized root Node.
func New[T any]() *Trie[T] {
	return &Trie[T]{
		root: &node[T]{}, // Lazy init children map
		size: 0,
	}
}

// AllKeyValuesIter returns a sequence of all key-value pairs in the trie.
//
// The trie's read lock is held for the duration of the iteration, so the
// consumer must not call Add or Remove on this trie from inside the loop;
// doing so deadlocks. Break out first, or use AllKeyValues for a snapshot.
func (t *Trie[T]) AllKeyValuesIter() iter.Seq2[string, T] {
	return func(yield func(string, T) bool) {
		t.mu.RLock()
		defer t.mu.RUnlock()

		collectIter(t.root)(yield)
	}
}

// AllKeyValues returns a map of all key-value pairs in the trie.
func (t *Trie[T]) AllKeyValues() map[string]T {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return maps.Collect(collectIter(t.root))
}

// Add adds the key to the Trie, including meta data.
func (t *Trie[T]) Add(key string, meta T) *node[T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	if key == "" {
		return nil
	}

	nd := t.root
	nd.mask |= maskstring(key)

	// Segments are sliced out of the key, so they share its backing array
	// instead of allocating a fresh string per node.
	remaining := key

	for len(remaining) > 0 {
		firstRune, _ := utf8.DecodeRuneInString(remaining)

		// Check if there's a child starting with this rune. A nil children map
		// simply misses, which is the "no children yet" case.
		child, exists := nd.children[firstRune]
		if !exists {
			// No child with this first rune, create new one
			t.size++
			return nd.newChild(remaining, meta, key)
		}

		// Find common prefix between remaining and child's segment
		commonLen := commonPrefixLen(remaining, child.segment)

		if commonLen == len(child.segment) {
			// Full match with child's segment, continue down
			remaining = remaining[commonLen:]
			nd = child

			if len(remaining) > 0 {
				nd.mask |= maskstring(remaining)
				continue
			}

			// Key ends exactly at this node
			nd.meta = meta
			if nd.path != nil {
				// Duplicate key, already counted
				return nd
			}
			nd.path = &key
			t.size++
			return nd
		}

		// Partial match - need to split the child node
		// Create intermediate node with common prefix
		intermediate := &node[T]{
			segment:  child.segment[:commonLen],
			parent:   nd,
			children: make(map[rune]*node[T]),
		}

		// Update child's segment to be the non-common part
		child.segment = child.segment[commonLen:]
		child.parent = intermediate
		childFirstRune, _ := utf8.DecodeRuneInString(child.segment)
		intermediate.children[childFirstRune] = child

		// Update parent's children map
		nd.children[firstRune] = intermediate

		// Update masks. The child's segment shrank to the remainder, so its
		// mask has to be rebuilt before the intermediate can fold it in.
		// The intermediate must cover the common prefix it now owns as well:
		// omitting it makes the fuzzy search prune valid subtrees.
		child.recomputeMask()
		intermediate.recomputeMask()

		remaining = remaining[commonLen:]
		nd = intermediate

		if len(remaining) == 0 {
			// Key ends at the split point. The intermediate node was just
			// created, so this is always a new key, never a duplicate.
			nd.meta = meta
			nd.path = &key
			t.size++
			return nd
		}

		// Create new child for remaining part
		nd.mask |= maskstring(remaining)
		t.size++
		return nd.newChild(remaining, meta, key)
	}

	// Should not reach here
	return nd
}

// commonPrefixLen returns the length in bytes of the longest common prefix of
// a and b. Both are assumed to be valid UTF-8, and the result is always a rune
// boundary: a multi-byte rune whose leading bytes agree must not be split, or
// the segments either side of it stop being valid UTF-8.
func commonPrefixLen(a, b string) int {
	n := min(len(a), len(b))

	i := 0
	for i < n && a[i] == b[i] {
		i++
	}

	// Back up to the start of a partially matched rune. Position len(a) is
	// always a boundary, since a is valid UTF-8 on its own.
	for i < len(a) && !utf8.RuneStart(a[i]) {
		i--
	}
	return i
}

// Find finds and returns meta data associated
// with `key`.
func (t *Trie[T]) Find(key string) (*node[T], bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	nd := findNode(t.root, key, true)
	if nd == nil || nd.path == nil {
		return nil, false
	}

	return nd, true
}

func (t *Trie[T]) HasKeysWithPrefix(key string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	nd := findNode(t.root, key, false)
	return nd != nil
}

// Remove removes a key from the trie, ensuring that
// all bitmasks up to root are appropriately recalculated.
func (t *Trie[T]) Remove(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	nd := findNode(t.root, key, true)
	if nd == nil || nd.path == nil {
		return
	}

	t.size--

	// Mark node as non-terminal
	nd.path = nil

	// If node has children, we can't remove it, just mark as non-terminal
	if len(nd.children) > 0 {
		return
	}

	// Node has no children, we can remove it
	// Walk up and remove nodes that are no longer needed
	for nd != nil && nd.path == nil && len(nd.children) == 0 {
		parent := nd.parent
		if parent == nil {
			break
		}

		// Remove this node from parent's children
		if len(nd.segment) > 0 {
			firstRune, _ := utf8.DecodeRuneInString(nd.segment)
			delete(parent.children, firstRune)
		}

		// If parent now has only one child and is not terminal, we could merge
		// but we'll keep it simple for now
		nd = parent
	}

	// Recalculate bitmasks from this point up
	for n := nd; n != nil; n = n.parent {
		n.recomputeMask()
	}
}

// Keys returns all the keys currently stored in the trie.
func (t *Trie[T]) Keys() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.size == 0 {
		return []string{}
	}

	// Collect directly rather than delegating to PrefixSearch: the public
	// search methods take the read lock themselves, and sync.RWMutex read
	// locks are not reentrant, so nesting them deadlocks whenever a writer
	// queues up in between.
	keys := make([]string, 0, t.size)
	for key := range collectIter(t.root) {
		keys = append(keys, key)
	}
	return keys
}

// FuzzySearch performs a fuzzy search against the keys in the trie.
// FuzzySearch performs a fuzzy search against the keys in the trie, returning all keys
// with the given prefix. Results are returned sorted.
func (t *Trie[T]) FuzzySearch(pre string) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	keys := make([]string, 0, t.size)
	for key := range fuzzycollectIter(t.root, []rune(pre)) {
		keys = append(keys, key)
	}
	sort.Sort(ByKeys(keys))
	return keys
}

// FuzzySearchIter performs a fuzzy search and returns an iterator over matching keys.
// Unlike FuzzySearch, the keys are not sorted - they are yielded as they are found.
// This provides lazy evaluation and is more memory efficient for large result sets.
//
// The trie's read lock is held for the duration of the iteration, so the
// consumer must not call Add or Remove on this trie from inside the loop;
// doing so deadlocks. Break out first, or use FuzzySearch for a snapshot.
func (t *Trie[T]) FuzzySearchIter(pre string) iter.Seq[string] {
	return func(yield func(string) bool) {
		t.mu.RLock()
		defer t.mu.RUnlock()

		fuzzycollectIter(t.root, []rune(pre))(yield)
	}
}

// PrefixSearch performs a prefix search against the keys in the trie.
func (t *Trie[T]) PrefixSearch(pre string) []string {
	// Use PrefixSearchIter internally to avoid code duplication
	var keys []string
	for key := range t.PrefixSearchIter(pre) {
		keys = append(keys, key)
	}
	return keys
}

// PrefixSearchIter performs a prefix search and returns an iterator over matching key-value pairs.
// Unlike PrefixSearch, this returns an iterator that yields both keys and their associated values.
// This provides lazy evaluation and is more memory efficient for large result sets.
//
// The trie's read lock is held for the duration of the iteration, so the
// consumer must not call Add or Remove on this trie from inside the loop;
// doing so deadlocks. Break out first, or use PrefixSearch for a snapshot.
func (t *Trie[T]) PrefixSearchIter(pre string) iter.Seq2[string, T] {
	return func(yield func(string, T) bool) {
		t.mu.RLock()
		defer t.mu.RUnlock()

		nd := findNode(t.root, pre, false)
		if nd == nil {
			// No node found, yield nothing
			return
		}

		collectIter(nd)(yield)
	}
}

// newChild creates and returns a pointer to a new child for the node.
func (n *node[T]) newChild(segment string, meta T, fullKey string) *node[T] {
	if len(segment) == 0 {
		return nil
	}

	bitmask := maskstring(segment)
	child := &node[T]{
		segment: segment,
		mask:    bitmask,
		meta:    meta,
		parent:  n,
		path:    &fullKey,
	}

	n.ensureChildren()
	firstRune, _ := utf8.DecodeRuneInString(segment)
	n.children[firstRune] = child
	n.mask |= bitmask
	return child
}

// Val returns the value of the node.
func (n *node[T]) Val() T {
	return n.meta
}

// ensureChildren lazily initializes the children map if needed
func (n *node[T]) ensureChildren() {
	if n.children == nil {
		n.children = make(map[rune]*node[T])
	}
}

// recomputeMask rebuilds n.mask from n's own segment plus its children's masks,
// which is the invariant the fuzzy search prunes against: a node's mask covers
// every rune appearing in the subtree rooted at it, its own segment included.
// The children's masks must already be correct.
func (n *node[T]) recomputeMask() {
	mask := maskstring(n.segment)
	for _, c := range n.children {
		mask |= c.mask
	}
	n.mask = mask
}

// findNode walks the trie looking for key. When exact is true the key must end
// precisely on a node boundary, which is what lookups like Find and Remove
// need. When exact is false a key that stops partway through a compressed
// segment still matches, which is what prefix queries need.
func findNode[T any](nd *node[T], key string, exact bool) *node[T] {
	if nd == nil {
		return nil
	}

	remaining := key

	for len(remaining) > 0 {
		firstRune, _ := utf8.DecodeRuneInString(remaining)
		child, exists := nd.children[firstRune]
		if !exists {
			return nil
		}

		// The key runs out partway through the segment. That is a hit only for
		// prefix queries; an exact lookup needs the key to consume the whole
		// segment, not just land somewhere inside it.
		if len(remaining) < len(child.segment) {
			if exact || !strings.HasPrefix(child.segment, remaining) {
				return nil
			}
			return child
		}

		if !strings.HasPrefix(remaining, child.segment) {
			return nil
		}

		// Segment matches, continue
		remaining = remaining[len(child.segment):]
		if len(remaining) == 0 {
			return child
		}
		nd = child
	}

	return nd
}

// maskstring creates a bitmask for the runes in s. Ranging a string decodes
// runes in place, so unlike maskruneslice this needs no []rune allocation.
//
//go:inline
func maskstring(s string) uint64 {
	var m uint64
	for _, r := range s {
		m |= uint64(1) << uint64(r-'a')
	}
	return m
}

// maskruneslice creates a bitmask for the given runes.
// Optimized to eliminate bounds checking and enable vectorization.
//
//go:inline
func maskruneslice(rs []rune) uint64 {
	// Use 4 accumulators for better instruction-level parallelism
	var m0, m1, m2, m3 uint64

	// Process 4 elements at a time using slice patterns for BCE
	for len(rs) >= 4 {
		// Compiler knows rs[:4] is safe when len(rs) >= 4
		// This pattern eliminates all bounds checks
		r := rs[:4:4] // Full slice expression prevents capacity growth

		// No bounds checks on these accesses
		m0 |= uint64(1) << uint64(r[0]-'a')
		m1 |= uint64(1) << uint64(r[1]-'a')
		m2 |= uint64(1) << uint64(r[2]-'a')
		m3 |= uint64(1) << uint64(r[3]-'a')

		rs = rs[4:]
	}

	// Handle remaining elements (0-3)
	// Process remaining with explicit length checks for BCE
	switch len(rs) {
	case 3:
		m0 |= uint64(1) << uint64(rs[0]-'a')
		m1 |= uint64(1) << uint64(rs[1]-'a')
		m2 |= uint64(1) << uint64(rs[2]-'a')
	case 2:
		m0 |= uint64(1) << uint64(rs[0]-'a')
		m1 |= uint64(1) << uint64(rs[1]-'a')
	case 1:
		m0 |= uint64(1) << uint64(rs[0]-'a')
	}

	// Combine all accumulators
	return m0 | m1 | m2 | m3
}

// collectIter returns an iterator over all key-value pairs starting from the given node
func collectIter[T any](nd *node[T]) iter.Seq2[string, T] {
	return func(yield func(string, T) bool) {
		childrenCount := 0
		if nd.children != nil {
			childrenCount = len(nd.children)
		}
		nodes := make([]*node[T], 1, childrenCount+1)
		nodes[0] = nd
		for len(nodes) > 0 {
			i := len(nodes) - 1
			n := nodes[i]
			nodes = nodes[:i]
			if n.children != nil {
				for _, c := range n.children {
					nodes = append(nodes, c)
				}
			}
			if n.path != nil {
				if !yield(*n.path, n.meta) {
					return
				}
			}
		}
	}
}

// fuzzycollectIter performs a fuzzy search and yields matching keys as an iterator
func fuzzycollectIter[T any](nd *node[T], partial []rune) iter.Seq[string] {
	return func(yield func(string) bool) {
		if len(partial) == 0 {
			// If no partial pattern, yield all keys from this node
			for key := range collectIter(nd) {
				if !yield(key) {
					return
				}
			}
			return
		}

		// Use stack-based traversal for fuzzy matching
		type potentialNode struct {
			idx  int
			node *node[T]
		}

		potential := make([]potentialNode, 1, 128)
		potential[0] = potentialNode{node: nd, idx: 0}

		for len(potential) > 0 {
			i := len(potential) - 1
			p := potential[i]
			potential = potential[:i]

			m := maskruneslice(partial[p.idx:])
			if (p.node.mask & m) != m {
				continue
			}

			// Check if any rune in segment matches current partial rune.
			// Ranging the string decodes runes without allocating.
			for _, r := range p.node.segment {
				if p.idx < len(partial) && r == partial[p.idx] {
					p.idx++
					if p.idx == len(partial) {
						// Found a match, yield all terminals from this subtree
						for key := range collectIter(p.node) {
							if !yield(key) {
								return
							}
						}
						break
					}
				}
			}

			if p.idx < len(partial) && p.node.children != nil {
				for _, c := range p.node.children {
					potential = append(potential, potentialNode{node: c, idx: p.idx})
				}
			}
		}
	}
}
