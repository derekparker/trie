[![Go Reference](https://pkg.go.dev/badge/github.com/derekparker/trie/v3.svg)](https://pkg.go.dev/github.com/derekparker/trie/v3)

# Trie

A generic Go trie (prefix tree) for fast prefix and fuzzy string searching,
optimized for memory efficiency and minimal allocations.

The tree is path-compressed: single-child chains collapse into one node holding
a whole string segment rather than a character, so a trie of long keys with few
branch points stays small. All exported methods are safe for concurrent use.

## Install

```
go get github.com/derekparker/trie/v3
```

`go.mod` declares `go 1.26`, so that is the minimum the toolchain will accept.

## Usage

Create a trie, choosing the type of the value stored alongside each key:

```Go
import "github.com/derekparker/trie/v3"

t := trie.New[int]()
```

The type parameter can be anything, including a struct or `any` if you want to
store heterogeneous values.

Add keys, along with the value to associate with them:

```Go
t.Add("foobar", 1)
```

Re-adding an existing key updates its value without growing the trie.
`Add("")` is a no-op that returns `nil`.

Find a key and read its value back with `Val()`:

```Go
node, ok := t.Find("foobar")
if ok {
    value := node.Val() // 1
}
```

`Find` is an exact match: it will not report `"foobar"` as a hit for `"foo"`.

Remove keys with:

```Go
t.Remove("foobar")
```

Prefix search returns every key beginning with the prefix:

```Go
t.PrefixSearch("foo") // ["foobar", ...]
```

Fast test for whether any key has a prefix, without collecting the matches:

```Go
t.HasKeysWithPrefix("foo")
```

Fuzzy search returns every key containing the pattern's runes in order, not
necessarily adjacent. Results are sorted shortest-first, ties broken
lexicographically:

```Go
t.FuzzySearch("fb") // matches "foobar"
```

Every key currently stored:

```Go
t.Keys()
```

Every key with its value, as a map:

```Go
t.AllKeyValues()
```

### Iterators

Each search has a lazy counterpart that yields results as it finds them rather
than building a slice. These are worth reaching for when the result set is
large or when you plan to stop early:

```Go
for key, value := range t.PrefixSearchIter("foo") {
    fmt.Println(key, value)
}

for key := range t.FuzzySearchIter("fb") {
    fmt.Println(key)
}

for key, value := range t.AllKeyValuesIter() {
    fmt.Println(key, value)
}
```

Unlike the eager variants, the iterators do not sort — keys are yielded in the
order they are found.

**One caveat:** the iterators hold the trie's read lock for the duration of the
iteration, not just while the iterator is constructed. Do not call `Add` or
`Remove` on the same trie from inside the loop; that self-deadlocks. Break out
of the loop first, or use the eager `PrefixSearch` / `FuzzySearch` /
`AllKeyValues`, which return a snapshot you can iterate freely.

## Contributing

Fork this repo and run the tests with:

	go test -race

The `-race` flag matters: several tests exist specifically to catch data races
and deadlocks and pass trivially without it.

Benchmarks:

	go test -bench=. -benchmem

The build and prefix-search benchmarks read `/usr/share/dict/words` (~236k
entries) rather than the tiny `fixtures/test.txt`. They abort the benchmark run
outright on a machine that lacks that file.

Create a feature branch, write your tests and code, and submit a pull request.

## License

MIT
