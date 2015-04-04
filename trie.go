// Implementation of an R-Way Trie data structure.
//
// A Trie has a root Node which is the base of the tree.
// Each subsequent Node has a letter and children, which are
// nodes that have letter values associated with them.
package trie

import (
	"sort"
)

type Node struct {
	val      rune
	term     bool
	meta     interface{}
	mask     uint64
	parent   *Node
	children map[rune]*Node
}

type Trie struct {
	root *Node
	size int
}

type ByKeys []string

func (a ByKeys) Len() int           { return len(a) }
func (a ByKeys) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKeys) Less(i, j int) bool { return len(a[i]) < len(a[j]) }

const nul = 0x0

// Creates a new Trie with an initialized root Node.
func New() *Trie {
	node := newNode(nil, 0, 0, false)
	return &Trie{
		root: node,
		size: 0,
	}
}

// Returns the root node for the Trie.
func (t *Trie) Root() *Node {
	return t.root
}

// Adds the key to the Trie, including meta data. Meta data
// is stored as `interface{}` and must be type cast by
// the caller.
func (t *Trie) Add(key string, meta interface{}) *Node {
	t.size++
	runes := []rune(key)
	node := t.addrune(t.Root(), runes, 0)
	node.meta = meta
	return node
}

// Finds and returns meta data associated
// with `key`.
func (t *Trie) Find(key string) (*Node, bool) {
	node := t.nodeAtPath(key)
	if node == nil {
		return nil, false
	}
	node = node.Children()[nul]

	if !node.term {
		return nil, false
	}

	return node, true
}

// Removes a key from the trie, ensuring that
// all bitmasks up to root are appropriately recalculated.
func (t *Trie) Remove(key string) {
	var (
		i    int
		rs   = []rune(key)
		node = t.nodeAtPath(key)
	)

	t.size--
	for n := node.Parent(); n != nil; n = n.Parent() {
		i++
		if len(n.Children()) > 1 {
			r := rs[len(rs)-i]
			n.RemoveChild(r)
			break
		}
	}
}

// Returns all the keys currently stored in the trie.
func (t *Trie) Keys() []string {
	return t.PrefixSearch("")
}

// Performs a fuzzy search against the keys in the trie.
func (t Trie) FuzzySearch(pre string) []string {
	var (
		keys []string
		pm   []rune
	)

	fuzzycollect(t.Root(), 0, pm, []rune(pre), &keys)
	sort.Sort(ByKeys(keys))

	return keys
}

// Performs a prefix search against the keys in the trie.
func (t Trie) PrefixSearch(pre string) []string {
	var keys []string

	node := t.nodeAtPath(pre)
	if node == nil {
		return keys
	}

	collect(node, []rune(pre), &keys)
	return keys
}

func (t Trie) nodeAtPath(pre string) *Node {
	runes := []rune(pre)
	return findNode(t.Root(), runes)
}

func (t Trie) addrune(node *Node, runes []rune, i int) *Node {
	if len(runes) == 0 {
		return node.NewChild(0, 0, nul, true)
	}

	r := runes[0]
	c := node.Children()

	n, ok := c[r]
	bitmask := maskruneslice(runes)
	if !ok {
		n = node.NewChild(r, bitmask, r, false)
	}
	n.mask |= bitmask

	i++
	return t.addrune(n, runes[1:], i)
}

func newNode(parent *Node, val rune, m uint64, term bool) *Node {
	return &Node{
		val:      val,
		mask:     m,
		term:     term,
		parent:   parent,
		children: make(map[rune]*Node),
	}
}

// Creates and returns a pointer to a new child for the node.
func (n *Node) NewChild(r rune, bitmask uint64, val rune, term bool) *Node {
	node := newNode(n, val, bitmask, term)
	n.children[r] = node
	return node
}

func (n *Node) RemoveChild(r rune) {
	delete(n.children, r)

	n.recalculateMask()
	for parent := n.Parent(); parent != nil; parent = parent.Parent() {
		parent.recalculateMask()
	}
}

func (n *Node) recalculateMask() {
	n.mask = maskrune(n.Val())
	for k, c := range n.Children() {
		n.mask |= (maskrune(k) | c.Mask())
	}
}

// Returns the parent of this node.
func (n Node) Parent() *Node {
	return n.parent
}

// Returns the meta information of this node.
func (n Node) Meta() interface{} {
	return n.meta
}

// Returns the children of this node.
func (n Node) Children() map[rune]*Node {
	return n.children
}

func (n Node) Val() rune {
	return n.val
}

// Returns a uint64 representing the current
// mask of this node.
func (n Node) Mask() uint64 {
	return n.mask
}

func findNode(node *Node, runes []rune) *Node {
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
		m |= maskrune(r)
	}

	return m
}

func maskrune(r rune) uint64 {
	i := uint64(1)
	return i << (uint64(r) - 97)
}

func collect(node *Node, pre []rune, keys *[]string) {
	children := node.Children()
	for r, n := range children {
		if n.term {
			*keys = append(*keys, string(pre))
			continue
		}

		npre := append(pre, r)
		collect(n, npre, keys)
	}
}

func fuzzycollect(node *Node, idx int, partialmatch, partial []rune, keys *[]string) {
	var (
		m          uint64
		partiallen = len(partial)
	)

	if partiallen == idx {
		collect(node, partialmatch, keys)
		return
	}

	children := node.Children()
	m = maskruneslice(partial[idx:])
	for v, n := range children {
		if (n.mask & m) != m {
			continue
		}

		if v == partial[idx] {
			fuzzycollect(n, idx+1, append(partialmatch, v), partial, keys)
			continue
		}

		fuzzycollect(n, idx, append(partialmatch, v), partial, keys)
	}
}
