package trie

import (
	"bufio"
	"io"
	"log"
	"os"
	"strings"
)

// Implementation of an R-Way Trie data structure.
//
// A Trie has a root Node which is the base of the tree.
// Each subsequent Node has a letter and children, which are
// nodes that have letter values associated with them.

type Node struct {
	val      int
	children map[rune]*Node
}

type Trie struct {
	root *Node
	size int
}

const nul = 0x0

func newNode(val int) *Node {
	return &Node{
		val:      val,
		children: make(map[rune]*Node),
	}
}

func (n *Node) NewChild(r rune, val int) *Node {
	node := newNode(val)
	n.children[r] = node
	return node
}

// Returns the children of this node.
func (n Node) Children() map[rune]*Node {
	return n.children
}

func (n Node) Val() int {
	return n.val
}

// Creates a new Trie with an initialized root Node.
func CreateTrie() *Trie {
	node := newNode(0)
	t := &Trie{
		root: node,
		size: 0,
	}
	return t
}

// Returns the root node for the Trie.
func (t *Trie) Root() *Node {
	return t.root
}

// Adds the key to the Trie.
func (t *Trie) AddKey(key string) int {
	t.size++
	runes := []rune(key)
	return t.addrune(t.Root(), runes, 0)
}

func (t *Trie) AddKeysFromFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	reader := bufio.NewReader(file)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}

		line = strings.TrimSuffix(line, "\n")
		t.AddKey(line)
	}
}

func (t *Trie) Keys() []string {
	return t.KeysWithPrefix("")
}

func (t Trie) KeysWithPrefix(pre string) []string {
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
	return findNode(t.Root(), runes, 0)
}

func findNode(node *Node, runes []rune, d int) *Node {
	if node == nil {
		return nil
	}

	if len(runes) == 0 {
		return node
	}

	upper := len(runes)
	if d == upper {
		return node
	}

	n, ok := node.Children()[runes[d]]
	if !ok {
		return nil
	}

	d++
	return findNode(n, runes, d)
}

func (t Trie) addrune(node *Node, runes []rune, i int) int {
	if len(runes) == 0 {
		node.NewChild(nul, t.size)
		return i
	}

	r := runes[0]
	c := node.Children()

	n, ok := c[r]
	if !ok {
		n = node.NewChild(r, 0)
	}

	i++
	return t.addrune(n, runes[1:], i)
}

func collect(node *Node, pre []rune, keys *[]string) {
	if len(node.Children()) < 1 {
		return
	}

	for k, v := range node.Children() {
		if v.Val() > 0 {
			*keys = append(*keys, string(pre))
			continue
		}

		npre := append(pre, k)
		collect(v, npre, keys)
	}
}
