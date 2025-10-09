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
	"sync"
)

type node[T any] struct {
	mask     uint64
	parent   *node[T]
	children map[rune]*node[T] // keyed by first rune of child's segment
	meta     T
	path     *string // pointer to full key for terminal nodes

	segment   string // the string segment stored in this node
	depth     int32
	termCount int32
}

// Trie is a data structure that stores a set of strings.
type Trie[T any] struct {
	mu   sync.RWMutex
	root *node[T]
	size int
}

type ByKeys []string

func (a ByKeys) Len() int           { return len(a) }
func (a ByKeys) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKeys) Less(i, j int) bool { return len(a[i]) < len(a[j]) }

// New creates a new Trie with an initialized root Node.
func New[T any]() *Trie[T] {
	return &Trie[T]{
		root: &node[T]{depth: 0}, // Lazy init children map
		size: 0,
	}
}

// AllKeyValuesIter returns a sequence of all key-value pairs in the trie.
func (t *Trie[T]) AllKeyValuesIter() iter.Seq2[string, T] {
	return collectIter(t.root)
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

	t.size++
	keyRunes := []rune(key)
	bitmask := maskruneslice(keyRunes)
	nd := t.root
	nd.mask |= bitmask
	nd.termCount++

	remainingRunes := keyRunes

	for len(remainingRunes) > 0 {
		firstRune := remainingRunes[0]

		// Check if there's a child starting with this rune
		if len(nd.children) == 0 {
			// No children, create new child with full remaining string
			return nd.newChild(string(remainingRunes), meta, key)
		}

		child, exists := nd.children[firstRune]
		if !exists {
			// No child with this first rune, create new one
			return nd.newChild(string(remainingRunes), meta, key)
		}

		// Find common prefix between remaining and child's segment
		segmentRunes := []rune(child.segment)
		commonLen := commonPrefixLenRunes(remainingRunes, segmentRunes)

		if commonLen == len(segmentRunes) {
			// Full match with child's segment, continue down
			remainingRunes = remainingRunes[commonLen:]
			nd = child

			if len(remainingRunes) > 0 {
				bitmask := maskruneslice(remainingRunes)
				nd.mask |= bitmask
			}
			nd.termCount++

			if len(remainingRunes) == 0 {
				// Key ends exactly at this node
				nd.meta = meta
				if nd.path == nil {
					nd.path = &key
				}
				return nd
			}
			continue
		}

		// Partial match - need to split the child node
		// Create intermediate node with common prefix
		intermediate := &node[T]{
			segment:   string(segmentRunes[:commonLen]),
			parent:    nd,
			depth:     nd.depth + 1,
			children:  make(map[rune]*node[T]),
			termCount: child.termCount,
		}

		// Update child's segment to be the non-common part
		childNewSegmentRunes := segmentRunes[commonLen:]
		child.segment = string(childNewSegmentRunes)
		child.parent = intermediate
		intermediate.children[childNewSegmentRunes[0]] = child

		// Update parent's children map
		nd.children[firstRune] = intermediate

		// Update masks
		if child.children != nil {
			for _, c := range child.children {
				intermediate.mask |= c.mask
			}
		}
		if len(childNewSegmentRunes) > 0 {
			intermediate.mask |= maskruneslice(childNewSegmentRunes)
		}

		remainingRunes = remainingRunes[commonLen:]
		nd = intermediate

		if len(remainingRunes) > 0 {
			bitmask := maskruneslice(remainingRunes)
			nd.mask |= bitmask
		}
		nd.termCount++

		if len(remainingRunes) == 0 {
			// Key ends at the split point
			nd.meta = meta
			nd.path = &key
			return nd
		}

		// Create new child for remaining part
		newChild := nd.newChild(string(remainingRunes), meta, key)
		return newChild
	}

	// Should not reach here
	return nd
}

// commonPrefixLenRunes returns the length of the common prefix between two rune slices
func commonPrefixLenRunes(r1, r2 []rune) int {
	minLen := min(len(r1), len(r2))

	for i := range minLen {
		if r1[i] != r2[i] {
			return i
		}
	}
	return minLen
}

// Find finds and returns meta data associated
// with `key`.
func (t *Trie[T]) Find(key string) (*node[T], bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	nd := findNode(t.root, key)
	if nd == nil || nd.path == nil {
		return nil, false
	}

	return nd, true
}

func (t *Trie[T]) HasKeysWithPrefix(key string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	nd := findNode(t.root, key)
	return nd != nil
}

// Remove removes a key from the trie, ensuring that
// all bitmasks up to root are appropriately recalculated.
func (t *Trie[T]) Remove(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	nd := findNode(t.root, key)
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
			firstRune := []rune(nd.segment)[0]
			delete(parent.children, firstRune)
		}

		// If parent now has only one child and is not terminal, we could merge
		// but we'll keep it simple for now
		nd = parent
	}

	// Recalculate bitmasks from this point up
	for n := nd; n != nil; n = n.parent {
		n.mask = 0
		if n.children != nil {
			for _, c := range n.children {
				n.mask |= c.mask
			}
		}
		if len(n.segment) > 0 {
			n.mask |= maskruneslice([]rune(n.segment))
		}
	}
}

// Keys returns all the keys currently stored in the trie.
func (t *Trie[T]) Keys() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.size == 0 {
		return []string{}
	}

	return t.PrefixSearch("")
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
func (t *Trie[T]) FuzzySearchIter(pre string) iter.Seq[string] {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return fuzzycollectIter(t.root, []rune(pre))
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
func (t *Trie[T]) PrefixSearchIter(pre string) iter.Seq2[string, T] {
	t.mu.RLock()
	defer t.mu.RUnlock()

	nd := findNode(t.root, pre)
	if nd == nil {
		// Return an empty iterator if no node is found
		return func(yield func(string, T) bool) {}
	}

	return collectIter(nd)
}

// newChild creates and returns a pointer to a new child for the node.
func (n *node[T]) newChild(segment string, meta T, fullKey string) *node[T] {
	runes := []rune(segment)
	if len(runes) == 0 {
		return nil
	}

	bitmask := maskruneslice(runes)
	child := &node[T]{
		segment: segment,
		mask:    bitmask,
		meta:    meta,
		parent:  n,
		depth:   n.depth + 1,
		path:    &fullKey,
	}

	n.ensureChildren()
	firstRune := runes[0]
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

func findNode[T any](nd *node[T], key string) *node[T] {
	if nd == nil {
		return nil
	}

	remaining := key

	for len(remaining) > 0 {
		if len(nd.children) == 0 {
			return nil
		}

		runes := []rune(remaining)
		firstRune := runes[0]

		child, exists := nd.children[firstRune]
		if !exists {
			return nil
		}

		// Check if remaining matches child's segment
		segmentRunes := []rune(child.segment)
		remainingRunes := []rune(remaining)

		// For prefix search: allow partial match if remaining is shorter
		matchLen := min(len(segmentRunes), len(remainingRunes))

		// Compare segment with beginning of remaining
		for i := range matchLen {
			if remainingRunes[i] != segmentRunes[i] {
				return nil
			}
		}

		// If we've consumed all of remaining, this is the node we want
		if len(remainingRunes) <= len(segmentRunes) {
			return child
		}

		// Segment matches, continue
		remaining = string(remainingRunes[len(segmentRunes):])
		nd = child
	}

	return nd
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

type potentialSubtree[T any] struct {
	idx  int
	node *node[T]
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

			// Check if any rune in segment matches current partial rune
			segmentRunes := []rune(p.node.segment)
			for _, r := range segmentRunes {
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
