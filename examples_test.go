package deepdiff

import (
	"encoding/json"
	"fmt"
)

func ExampleDiffJSON() {
	// start with two slightly different json documents
	aJSON := []byte(`{
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

	bJSON := []byte(`{
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
	diffs, err := Diff(a, b)
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
