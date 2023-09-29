// Implementation of an R-Way Trie data structure.
//
// A Trie has a root Node which is the base of the tree.
// Each subsequent Node has a letter and children, which are
// nodes that have letter values associated with them.
package trie

import (
	"sort"
	"sync"
)

type node[T any] struct {
	val       rune
	path      string
	term      bool
	depth     int
	meta      T
	mask      uint64
	parent    *node[T]
	children  map[rune]*node[T]
	termCount int
}

type Trie[T any] struct {
	mu   sync.RWMutex
	root *node[T]
	size int
}

type ByKeys []string

func (a ByKeys) Len() int           { return len(a) }
func (a ByKeys) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKeys) Less(i, j int) bool { return len(a[i]) < len(a[j]) }

const nul = 0x0

// New creates a new Trie with an initialized root Node.
func New[T any]() *Trie[T] {
	return &Trie[T]{
		root: &node[T]{children: make(map[rune]*node[T]), depth: 0},
		size: 0,
	}
}

// Add adds the key to the Trie, including meta data. Meta data
// is stored as `interface{}` and must be type cast by
// the caller.
func (t *Trie[T]) Add(key string, meta T) *node[T] {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.size++
	runes := []rune(key)
	bitmask := maskruneslice(runes)
	nd := t.root
	nd.mask |= bitmask
	nd.termCount++
	for i := range runes {
		r := runes[i]
		bitmask = maskruneslice(runes[i:])
		if n, ok := nd.children[r]; ok {
			nd = n
			nd.mask |= bitmask
		} else {
			nd = nd.newEmptyChild(r, "", bitmask)
		}
		nd.termCount++
	}
	nd = nd.newChild(nul, key, 0, meta, true)

	return nd
}

// Find finds and returns meta data associated
// with `key`.
func (t *Trie[T]) Find(key string) (*node[T], bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	nd := findNode(t.root, []rune(key))
	if nd == nil {
		return nil, false
	}

	nd, ok := nd.children[nul]
	if !ok || !nd.term {
		return nil, false
	}

	return nd, true
}

func (t *Trie[T]) HasKeysWithPrefix(key string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	nd := findNode(t.root, []rune(key))
	return nd != nil
}

// Remove removes a key from the trie, ensuring that
// all bitmasks up to root are appropriately recalculated.
func (t *Trie[T]) Remove(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	var (
		rs = []rune(key)
		nd = findNode(t.root, []rune(key))
	)

	if nd == nil {
		return
	}

	t.size--
	for n := nd.parent; n != nil; n = n.parent {
		if n == t.root {
			t.root = &node[T]{children: make(map[rune]*node[T])}
			break
		}

		if len(n.children) > 1 {
			n.removeChild(rs[n.depth])
			break
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
func (t *Trie[T]) FuzzySearch(pre string) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	keys := fuzzycollect(t.root, []rune(pre))
	sort.Sort(ByKeys(keys))
	return keys
}

// PrefixSearch performs a prefix search against the keys in the trie.
func (t *Trie[T]) PrefixSearch(pre string) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	nd := findNode(t.root, []rune(pre))
	if nd == nil {
		return nil
	}

	return collect(nd)
}

// newChild creates and returns a pointer to a new child for the node.
func (n *node[T]) newChild(val rune, path string, bitmask uint64, meta T, term bool) *node[T] {
	node := &node[T]{
		val:      val,
		path:     path,
		mask:     bitmask,
		term:     term,
		meta:     meta,
		parent:   n,
		children: make(map[rune]*node[T]),
		depth:    n.depth + 1,
	}
	n.children[node.val] = node
	n.mask |= bitmask
	return node
}

// newEmptyChild creates and returns a pointer to a new child for the node.
func (n *node[T]) newEmptyChild(val rune, path string, bitmask uint64) *node[T] {
	node := &node[T]{
		val:      val,
		path:     path,
		mask:     bitmask,
		parent:   n,
		children: make(map[rune]*node[T]),
		depth:    n.depth + 1,
	}
	n.children[node.val] = node
	n.mask |= bitmask
	return node
}

func (n *node[T]) removeChild(r rune) {
	delete(n.children, r)
	for nd := n.parent; nd != nil; nd = nd.parent {
		nd.mask ^= nd.mask
		nd.mask |= uint64(1) << uint64(nd.val-'a')
		for _, c := range nd.children {
			nd.mask |= c.mask
		}
	}
}

func findNode[T any](nd *node[T], runes []rune) *node[T] {
	if nd == nil {
		return nil
	}

	if len(runes) == 0 {
		return nd
	}

	n, ok := nd.children[runes[0]]
	if !ok {
		return nil
	}

	var nrunes []rune
	if len(runes) > 1 {
		nrunes = runes[1:]
	} else {
		nrunes = runes[0:0]
	}

	return findNode(n, nrunes)
}

func maskruneslice(rs []rune) uint64 {
	var m uint64
	for _, r := range rs {
		m |= uint64(1) << uint64(r-'a')
	}
	return m
}

func collect[T any](nd *node[T]) []string {
	keys := make([]string, 0, nd.termCount)
	nodes := make([]*node[T], 1, len(nd.children)+1)
	nodes[0] = nd
	for len(nodes) > 0 {
		i := len(nodes) - 1
		n := nodes[i]
		nodes = nodes[:i]
		for _, c := range n.children {
			nodes = append(nodes, c)
		}
		if n.term {
			word := n.path
			keys = append(keys, word)
		}
	}
	return keys
}

type potentialSubtree[T any] struct {
	idx  int
	node *node[T]
}

func fuzzycollect[T any](nd *node[T], partial []rune) (keys []string) {
	if len(partial) == 0 {
		return collect(nd)
	}

	potential := []potentialSubtree[T]{{node: nd, idx: 0}}
	for len(potential) > 0 {
		i := len(potential) - 1
		p := potential[i]
		potential = potential[:i]
		m := maskruneslice(partial[p.idx:])
		if (p.node.mask & m) != m {
			continue
		}

		if p.node.val == partial[p.idx] {
			p.idx++
			if p.idx == len(partial) {
				keys = append(keys, collect(p.node)...)
				continue
			}
		}

		for _, c := range p.node.children {
			potential = append(potential, potentialSubtree[T]{node: c, idx: p.idx})
		}
	}
	return keys
}
