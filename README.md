# deepdiff
[![Qri](https://img.shields.io/badge/made%20by-qri-magenta.svg?style=flat-square)](https://qri.io)
[![GoDoc](https://godoc.org/github.com/qri-io/deepdiff?status.svg)](http://godoc.org/github.com/qri-io/deepdiff)
[![License](https://img.shields.io/github/license/qri-io/deepdiff.svg?style=flat-square)](./LICENSE)
[![Codecov](https://img.shields.io/codecov/c/github/qri-io/deepdiff.svg?style=flat-square)](https://codecov.io/gh/qri-io/deepdiff)
[![CI](https://img.shields.io/circleci/project/github/qri-io/deepdiff.svg?style=flat-square)](https://circleci.com/gh/qri-io/deepdiff)
[![Go Report Card](https://goreportcard.com/badge/github.com/qri-io/deepdiff)](https://goreportcard.com/report/github.com/qri-io/deepdiff)

deepdiff is a structured data differ that aims for near-linear time complexity. It's intended to calculate differences & apply patches to structured data ranging from  0-500MBish of encoded JSON.

Diffing structured data carries additional complexity when compared to the standard unix diff utility, which operates on lines of text. By using the structure of data itself, deepdiff is able to provide a rich description of changes that maps onto the structure of the data itself. deepdiff ignores semantically irrelevant changes like whitespace, and can isolate changes like column changes to tabular data to only the relevant switches

Most algorithms in this space have quadratic time complexity, which from our testing makes them very slow on 3MB JSON documents and unable to complete on 5MB or more. deepdiff currently hovers around the 0.9Sec/MB range on 4 core processors

Instead of operating on JSON directly, deepdiff operates on document trees consisting of the go types created by unmarshaling from JSON, which are two complex types:
```
  map[string]interface{}
  []interface{}
```
and five scalar types:
```
  string, int, float64, bool, nil
```

By operating on native go types deepdiff can compare documents encoded in different formats, for example decoded CSV or CBOR.

deepdiff is based off an algorithm designed for diffing XML documents outlined in
[_Detecting Changes in XML Documents by Grégory Cobéna & Amélie Marian_](https://ieeexplore.ieee.org/document/994696)

It's been adapted to fit purposes of diffing for Qri: https://github.com/qri-io/qri, folding in parallelism primitives afforded by the go language

deepdiff also includes a tool for applying patches, see documentation for details.

## Project Status:

:construction_worker_woman: :construction_worker_man: This is a very new project that hasn't been properly vetted in testing enviornments. Issues/PRs welcome & appriciated. :construction_worker_woman: :construction_worker_man:

## Benchmarks

Run on a 4 core MacBook Pro:

```
$ go test -bench . --run XXX -v --benchmem
goos: darwin
goarch: amd64
pkg: github.com/qri-io/deepdiff
BenchmarkDiff1-4          	   20000	     88167 ns/op	   13324 B/op	     496 allocs/op
BenchmarkDiffDatasets-4   	    5000	    241119 ns/op	   53367 B/op	    1614 allocs/op
BenchmarkDiff5MB-4        	       1	4357009141 ns/op	783217944 B/op	29952860 allocs/op
PASS
ok  	github.com/qri-io/deepdiff	8.369s
```


### Getting Involved

We would love involvement from more people! If you notice any errors or would like to submit changes, please see our
[Contributing Guidelines](./.github/CONTRIBUTING.md).


## Basic Usage

Here's a quick example pulled from the [godoc](https://godoc.org/github.com/qri-io/deepdiff):

```go
package main

import (
	"encoding/json"
  "fmt"
  
  "github.com/qri-io/deepdiff"
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

	// Diff will use default configuration to produce a slice of Changes
	// by default Diff will not generate update change only inserts & deletes
	diffs, err := deepdiff.Diff(a, b)
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

## License

The deepdiff library is licensed under the [GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html)