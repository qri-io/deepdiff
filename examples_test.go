package deepdiff

import (
	"context"
	"encoding/json"
	"fmt"
)

func ExampleDiffJSON() {
	// we'll use the background as our execution context
	ctx := context.Background()

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

	// create a differ, using the default configuration
	dd := NewDeepDiff()

	// Diff will produce a slice of Deltas that describe the structured changes.
	// by default Diff will not calculate moves, only inserts, deletes, and
	// updates
	diffs, err := dd.Diff(ctx, a, b)
	if err != nil {
		panic(err)
	}

	// Format the changes for terminal output
	change, err := FormatPretty(diffs)
	if err != nil {
		panic(err)
	}

	fmt.Println(change)
	// Output: + a: 99
	// - a: 100
	// baz:
	//   + e: "thirty-thousand-something-dogecoin"
	//   + f: false
	//   - e: null
	//   - g: "apples-and-oranges"
	//   a:
	//     + b: 5
	//     - b: 4
}
