// Implementation of an R-Way Trie data structure.
//
// A Trie has a root Node which is the base of the tree.
// Each subsequent Node has a letter and children, which are
// nodes that have letter values associated with them.
package trie

import "sort"

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
		bitmask := maskruneslice(runes[i:])
		n, ok := node.children[r]
		if !ok {
			n = node.NewChild(r, bitmask, r, false)
		}
		node = n
		node.mask |= bitmask
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

func collect(node *Node, pre []rune, keys *[]string) {
	for r, n := range node.children {
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
		val        rune
		partiallen = len(partial)
	)

	if partiallen == idx {
		collect(node, partialmatch, keys)
		return
	}

	m = maskruneslice(partial[idx:])
	for v, n := range node.children {
		val = partial[idx]
		if (n.mask&m) != m && v != val {
			continue
		}

		if v == val {
			fuzzycollect(n, idx+1, append(partialmatch, v), partial, keys)
			continue
		}

		fuzzycollect(n, idx, append(partialmatch, v), partial, keys)
	}
}
