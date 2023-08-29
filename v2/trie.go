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

type Node[T any] struct {
	val       rune
	path      string
	term      bool
	depth     int
	meta      T
	mask      uint64
	parent    *Node[T]
	children  map[rune]*Node[T]
	termCount int
}

type Trie[T any] struct {
	mu   sync.Mutex
	root *Node[T]
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
		root: &Node[T]{children: make(map[rune]*Node[T]), depth: 0},
		size: 0,
	}
}

// Root returns the root node for the Trie.
func (t *Trie[T]) Root() *Node[T] {
	return t.root
}

// Add adds the key to the Trie, including meta data. Meta data
// is stored as `interface{}` and must be type cast by
// the caller.
func (t *Trie[T]) Add(key string, meta T) *Node[T] {
	t.mu.Lock()

	t.size++
	runes := []rune(key)
	bitmask := maskruneslice(runes)
	node := t.root
	node.mask |= bitmask
	node.termCount++
	for i := range runes {
		r := runes[i]
		bitmask = maskruneslice(runes[i:])
		if n, ok := node.children[r]; ok {
			node = n
			node.mask |= bitmask
		} else {
			node = node.NewEmptyChild(r, "", bitmask)
		}
		node.termCount++
	}
	node = node.NewChild(nul, key, 0, meta, true)
	t.mu.Unlock()

	return node
}

// Find finds and returns meta data associated
// with `key`.
func (t *Trie[T]) Find(key string) (*Node[T], bool) {
	node := findNode(t.Root(), []rune(key))
	if node == nil {
		return nil, false
	}

	node, ok := node.Children()[nul]
	if !ok || !node.term {
		return nil, false
	}

	return node, true
}

func (t *Trie[T]) HasKeysWithPrefix(key string) bool {
	node := findNode(t.Root(), []rune(key))
	return node != nil
}

// Remove removes a key from the trie, ensuring that
// all bitmasks up to root are appropriately recalculated.
func (t *Trie[T]) Remove(key string) {
	var (
		i    int
		rs   = []rune(key)
		node = findNode(t.Root(), []rune(key))
	)

	if node == nil {
		return
	}

	t.mu.Lock()

	t.size--
	for n := node.Parent(); n != nil; n = n.Parent() {
		i++

		if n == t.root {
			t.root = &Node[T]{children: make(map[rune]*Node[T])}
			break
		}

		if len(n.Children()) > 1 {
			r := rs[len(rs)-i]
			n.RemoveChild(r)
			break
		}
	}
	t.mu.Unlock()
}

// Keys returns all the keys currently stored in the trie.
func (t *Trie[T]) Keys() []string {
	if t.size == 0 {
		return []string{}
	}

	return t.PrefixSearch("")
}

// FuzzySearch performs a fuzzy search against the keys in the trie.
func (t *Trie[T]) FuzzySearch(pre string) []string {
	keys := fuzzycollect(t.Root(), []rune(pre))
	sort.Sort(ByKeys(keys))
	return keys
}

// PrefixSearch performs a prefix search against the keys in the trie.
func (t *Trie[T]) PrefixSearch(pre string) []string {
	node := findNode(t.Root(), []rune(pre))
	if node == nil {
		return nil
	}

	return collect(node)
}

// NewChild creates and returns a pointer to a new child for the node.
func (n *Node[T]) NewChild(val rune, path string, bitmask uint64, meta T, term bool) *Node[T] {
	node := &Node[T]{
		val:      val,
		path:     path,
		mask:     bitmask,
		term:     term,
		meta:     meta,
		parent:   n,
		children: make(map[rune]*Node[T]),
		depth:    n.depth + 1,
	}
	n.children[node.val] = node
	n.mask |= bitmask
	return node
}

// NewEmptyChild creates and returns a pointer to a new child for the node.
func (n *Node[T]) NewEmptyChild(val rune, path string, bitmask uint64) *Node[T] {
	node := &Node[T]{
		val:      val,
		path:     path,
		mask:     bitmask,
		parent:   n,
		children: make(map[rune]*Node[T]),
		depth:    n.depth + 1,
	}
	n.children[node.val] = node
	n.mask |= bitmask
	return node
}

func (n *Node[T]) RemoveChild(r rune) {
	delete(n.children, r)
	for nd := n.parent; nd != nil; nd = nd.parent {
		nd.mask ^= nd.mask
		nd.mask |= uint64(1) << uint64(nd.val-'a')
		for _, c := range nd.children {
			nd.mask |= c.mask
		}
	}
}

// Parent returns the parent of this node.
func (n *Node[T]) Parent() *Node[T] {
	return n.parent
}

// Meta returns the meta information of this node.
func (n *Node[T]) Meta() T {
	return n.meta
}

// Children returns the children of this node.
func (n *Node[T]) Children() map[rune]*Node[T] {
	return n.children
}

// Terminating returns true if this node terminates an entry in the Trie.
func (n *Node[T]) Terminating() bool {
	return n.term
}

// Val returns the rune value of the Node.
func (n *Node[T]) Val() rune {
	return n.val
}

// Depth returns this nodes depth in the tree.
func (n *Node[T]) Depth() int {
	return n.depth
}

// Mask returns a uint64 representing the current
// mask of this node.
func (n *Node[T]) Mask() uint64 {
	return n.mask
}

func findNode[T any](node *Node[T], runes []rune) *Node[T] {
	if node == nil {
		return nil
	}

	if len(runes) == 0 {
		return node
	}

	n, ok := node.Children()[runes[0]]
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

func collect[T any](node *Node[T]) []string {
	var (
		n *Node[T]
		i int
	)
	keys := make([]string, 0, node.termCount)
	nodes := make([]*Node[T], 1, len(node.children)+1)
	nodes[0] = node
	for l := len(nodes); l != 0; l = len(nodes) {
		i = l - 1
		n = nodes[i]
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
	node *Node[T]
}

func fuzzycollect[T any](node *Node[T], partial []rune) []string {
	if len(partial) == 0 {
		return collect(node)
	}

	var (
		m    uint64
		i    int
		p    potentialSubtree[T]
		keys []string
	)

	potential := []potentialSubtree[T]{potentialSubtree[T]{node: node, idx: 0}}
	for l := len(potential); l > 0; l = len(potential) {
		i = l - 1
		p = potential[i]
		potential = potential[:i]
		m = maskruneslice(partial[p.idx:])
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
