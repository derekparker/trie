package trie

import (
	"sync"
	"testing"
)

// TestReturnedNodeValConcurrentWithAdd exercises the race the README's
// concurrency guarantee forbids: a node returned by Add/Find is still
// reachable, and Add overwrites its meta while Val reads it. The per-node
// metaMu must make the pair safe.
func TestReturnedNodeValConcurrentWithAdd(t *testing.T) {
	tr := New[int]()
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

// TestReturnedNodeValDuringIteration checks that Val remains safe to call
// while the trie's read lock is held by an iterator, and that a removed node's
// value is still readable.
func TestReturnedNodeValDuringIteration(t *testing.T) {
	tr := New[int]()
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