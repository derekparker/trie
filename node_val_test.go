package trie_test

import (
	"sync"
	"testing"

	trie "github.com/derekparker/trie/v3"
)

func TestReturnedNodeValConcurrentWithAdd(t *testing.T) {
	tr := trie.New[int]()
	added := tr.Add("foobar", 0)
	found, ok := tr.Find("foobar")
	if !ok {
		t.Fatal("Find failed")
	}
	// Split the segment without invalidating the previously returned nodes.
	tr.Add("foobaz", 1)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			tr.Add("foobar", i)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			_ = added.Val()
			_ = found.Val()
		}
	}()
	wg.Wait()
	if got := found.Val(); got != 999 {
		t.Errorf("Val = %d, want 999", got)
	}
}

func TestReturnedNodeValDuringIteration(t *testing.T) {
	tr := trie.New[int]()
	n := tr.Add("foo", 1)
	for range tr.AllKeyValuesIter() {
		if got := n.Val(); got != 1 {
			t.Errorf("Val = %d, want 1", got)
		}
	}
	tr.Remove("foo")
	if got := n.Val(); got != 1 {
		t.Errorf("removed node Val = %d, want 1", got)
	}
}
