//go:build goexperiment.rangefunc

package trie

import "iter"

// All returns a sequence of all key-value pairs in the trie.
func (t *Trie[T]) All() iter.Seq2[string, T] {
	return func(yield func(string, T) bool) {
		nodes := make([]*node[T], 1, len(t.root.children)+1)
		nodes[0] = t.root
		for len(nodes) > 0 {
			i := len(nodes) - 1
			n := nodes[i]
			nodes = nodes[:i]
			for _, c := range n.children {
				nodes = append(nodes, c)
			}
			if n.term {
				yield(n.path, n.meta)
			}
		}
	}
}
