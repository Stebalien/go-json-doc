# go-json-doc

[![Build Status](https://travis-ci.org/Stebalien/go-json-doc.svg?branch=master)](https://travis-ci.org/Stebalien/go-json-doc)
[![codecov](https://codecov.io/gh/Stebalien/go-json-doc/branch/master/graph/badge.svg)](https://codecov.io/gh/Stebalien/go-json-doc)

> Go documentation generator for JSON structs.

go-json-doc uses reflection to generate JSON schemas for go structs.

## Usage

API documentation can be read at [Godoc](https://godoc.org/github.com/Stebalien/go-json-doc).

```go
doc, err := jsondoc.NewGlossary().Describe(new(struct{
  Name string
  Age int
  Occupation `json:"Job"`
}))

fmt.Println(doc)
// {
//   "Name": "<string>",
//   "Age": "<string>",
//   "Job": "<string>"
// }
```

A more complete example can be found in the
[examples](://github.com/Stebalien/go-json-doc/branch/master/example/)
directory.

## Contribute

PRs accepted.

## Contributors

* Steven Allen (@stebalien)
* Initial concept by Hector Sanjuan (@hsanjuan)

## License

MIT © Steven Allen
