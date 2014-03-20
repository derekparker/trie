# Trie
Data structure and relevant algorithms for extremely fast prefix/fuzzy string searching.

## Usage

Create a Trie with:

```Go
t := trie.NewTrie()
```

Add Keys with:

```Go
t.Add("foobar")
```

Remove Keys with:
```Go
t.Remove("foobar")
```

Prefix search with:

```Go
t.PrefixSearch("foo")
```

Fuzzy search with:

```Go
t.FuzzySearch("fb")
```

## Contributing
Fork this repo and run tests with:

	go test

Create a feature branch, write your tests and code and submit a pull request.

## License
MIT
