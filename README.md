# difff
[![Qri](https://img.shields.io/badge/made%20by-qri-magenta.svg?style=flat-square)](https://qri.io)
[![GoDoc](https://godoc.org/github.com/qri-io/difff?status.svg)](http://godoc.org/github.com/qri-io/difff)
[![License](https://img.shields.io/github/license/qri-io/difff.svg?style=flat-square)](./LICENSE)
[![Codecov](https://img.shields.io/codecov/c/github/qri-io/difff.svg?style=flat-square)](https://codecov.io/gh/qri-io/difff)
[![CI](https://img.shields.io/circleci/project/github/qri-io/difff.svg?style=flat-square)](https://circleci.com/gh/qri-io/difff)
[![Go Report Card](https://goreportcard.com/badge/github.com/qri-io/difff)](https://goreportcard.com/report/github.com/qri-io/difff)

difff (with an extra f) is a structured data differ that aims for near-linear time complexity. It's intended to calculate differences & apply patches to structured data ranging from  0-500MBish of encoded JSON.

Diffing structured data carries additional complexity when compared to the standard unix diff utility, which operates on lines of text. By using the structure of data itself, difff is able to provide a rich description of changes that maps onto the structure of the data itself. difff ignores semantically irrelevant changes like whitespace, and can isolate changes like column changes to tabular data to only the relevant switches

Most algorithms in this space have quadratic time complexity, which from our testing makes them very slow on 3MB JSON documents and unable to complete on 5MB or more. difff currently hovers around the 0.9Sec/MB range on 4 core processors

Instead of operating on JSON directly, difff operates on document trees consisting of the go types created by unmarshaling from JSON, which are two complex types:
```
  map[string]interface{}
  []interface{}
```
and five scalar types:
```
  string, int, float64, bool, nil
```

By operating on native go types difff can compare documents encoded in different formats, for example decoded CSV or CBOR.

difff is based off an algorithm designed for diffing XML documents outlined in
[_Detecting Changes in XML Documents by Grégory Cobéna & Amélie Marian_](https://ieeexplore.ieee.org/document/994696)

It's been adapted to fit purposes of diffing for Qri: https://github.com/qri-io/qri, folding in parallelism primitives afforded by the go language

Difff also includes a tool for applying patches, see documentation for details.

## Project Status:

:construction_worker_woman: :construction_worker_man: This is a very new project that hasn't been properly vetted in testing enviornments. Issues/PRs welcome & appriciated.

## Benchmarks

Run on a 4 core MacBook Pro:

```
$ go test -bench . --run XXX -v --benchmem
goos: darwin
goarch: amd64
pkg: github.com/qri-io/difff
BenchmarkDiff1-4          	   20000	     88167 ns/op	   13324 B/op	     496 allocs/op
BenchmarkDiffDatasets-4   	    5000	    241119 ns/op	   53367 B/op	    1614 allocs/op
BenchmarkDiff5MB-4        	       1	4357009141 ns/op	783217944 B/op	29952860 allocs/op
PASS
ok  	github.com/qri-io/difff	8.369s
```


### Getting Involved

We would love involvement from more people! If you notice any errors or would like to submit changes, please see our
[Contributing Guidelines](./.github/CONTRIBUTING.md).


## Basic Usage

Here's a quick example pulled from the [godoc](https://godoc.org/github.com/qri-io/difff):

```go
package main

import (
	"encoding/json"
  "fmt"
  
  "github.com/qri-io/difff"
)

// start with two slightly different json documents
var aJSON = []byte(`{
  "a": 100,
  "foo": [1,2,3],
  "bar": false,
  "baz": {
    "a": {
      "b": 4,
      "c": false,
      "d": "apples-and-oranges"
    },
    "e": null,
    "g": "apples-and-oranges"
  }
}`)

var bJSON = []byte(`{
  "a": 99,
  "foo": [1,2,3],
  "bar": false,
  "baz": {
    "a": {
      "b": 5,
      "c": false,
      "d": "apples-and-oranges"
    },
    "e": "thirty-thousand-something-dogecoin",
    "f": false
  }
}`)

func main() {
	// unmarshal the data into generic interfaces
	var a, b interface{}
	if err := json.Unmarshal(aJSON, &a); err != nil {
		panic(err)
	}
	if err := json.Unmarshal(bJSON, &b); err != nil {
		panic(err)
	}

	// Diff will use default configuration to produce a slice of Deltas
	// that describe the structured changes. by default Diff will not calculate
	// moves, only inserts, deletes, and updates
	diffs, err := difff.Diff(a, b)
	if err != nil {
		panic(err)
	}

	// Format the changes for terminal output
	change, err := FormatPretty(diffs)
	if err != nil {
		panic(err)
	}

	fmt.Println(change)
	// Output: baz:
	//   + f: false
	//   - g: "apples-and-oranges"
	//   a:
	//     ~ b: 5
	//   ~ e: "thirty-thousand-something-dogecoin"
	// ~ a: 99
}
```
