// Implementation of an R-Way Trie data structure.
//
// A Trie has a root Node which is the base of the tree.
// Each subsequent Node has a letter and children, which are
// nodes that have letter values associated with them.
package trie

import (
	"fmt"
	"sort"
)

type Node struct {
	val      rune
	term     bool
	depth    int
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

const (
	nul    = 0x0
	asciiA = 'a'
)

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
	node := t.root
	for i := range runes {
		r := runes[i]
		bitmask := maskruneslice(runes[i+1:])
		if n, ok := node.children[r]; ok {
			node = n
			node.mask |= bitmask
			continue
		}
		node = node.NewChild(r, bitmask, r, false)
	}
	node = node.NewChild(0, 0, nul, true)
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
	keys := fuzzycollect(t.Root(), 0, []rune(pre))
	sort.Sort(ByKeys(keys))
	return keys
}

// Performs a prefix search against the keys in the trie.
func (t Trie) PrefixSearch(pre string) []string {
	node := t.nodeAtPath(pre)
	if node == nil {
		return nil
	}

	return collect(node)
}

func (t Trie) nodeAtPath(pre string) *Node {
	runes := []rune(pre)
	return findNode(t.Root(), runes)
}

func newNode(parent *Node, val rune, m uint64, term bool) *Node {
	var depth int
	if parent == nil {
		depth = 0
	} else {
		depth = parent.depth + 1
	}
	return &Node{
		val:      val,
		mask:     m,
		term:     term,
		parent:   parent,
		children: make(map[rune]*Node),
		depth:    depth,
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
	for nd := n.parent; nd != nil; nd = nd.parent {
		nd.mask ^= nd.mask
		for rn, c := range nd.children {
			nd.mask |= ((uint64(1) << uint64(rn-asciiA)) | c.mask)
		}
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
		m |= uint64(1) << uint64(r-asciiA)
	}

	return m
}

func collect(node *Node) []string {
	var (
		k     []string
		n     *Node
		nodes []*Node
	)
	nodes = append(nodes, node)
	for len(nodes) != 0 {
		i := len(nodes) - 1
		n = nodes[i]
		nodes = nodes[:i]
		for _, c := range n.children {
			nodes = append(nodes, c)
		}
		if n.term {
			word := make([]rune, n.depth-1)
			for p := n.parent; p.depth != 0; p = p.parent {
				word[p.depth-1] = p.val
			}
			k = append(k, string(word))
		}
	}
	return k
}

type potentialSubtree struct {
	idx  int
	node *Node
}

func fuzzycollect(node *Node, iidx int, partial []rune) []string {
	var (
		m         uint64
		val       rune
		keys      []string
		potential []potentialSubtree
		nodes     []*Node
	)

	potential = append(potential, potentialSubtree{node: node, idx: 0})
	for len(potential) > 0 {
		i := len(potential) - 1
		p := potential[i]
		potential = potential[:i]
		idx := p.idx
		m = maskruneslice(partial[idx:])

		val = partial[idx]
		if (p.node.mask&m) != m && p.node.val != val {
			fmt.Println("continue")
			continue
		}
		if p.node.val == val {
			idx++
		}
		if idx == len(partial) {
			nodes = append(nodes, p.node)
			continue
		}
		for _, c := range p.node.children {
			potential = append(potential, potentialSubtree{node: c, idx: idx})
		}
	}

	for i := range nodes {
		keys = append(keys, collect(nodes[i])...)
	}
	return keys
}
