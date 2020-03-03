package deepdiff

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Example() {
	// we'll use the background as our execution context
	ctx := context.Background()

	// start with two slightly different json documents
	aJSON := []byte(`{
		"a": 100,
		"baz": {
			"a": {
				"d": "apples-and-oranges"
			}
		}
	}`)

	bJSON := []byte(`{
		"a": 99,
		"baz": {
			"a": {
				"d": "apples-and-oranges"
			},
			"e": "thirty-thousand-something-dogecoin"
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
	dd := New()

	// Diff will produce a slice of Deltas that describe the structured changes.
	// by default Diff will not calculate moves, only inserts, deletes, and
	// updates
	diffs, err := dd.Diff(ctx, a, b)
	if err != nil {
		panic(err)
	}

	// diffs use a custom compact JSON Marshaller
	output, err := json.MarshalIndent(diffs, "", "  ")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(output))
	// Output:
	// [
	//   [
	//     "-",
	//     "a",
	//     100
	//   ],
	//   [
	//     "+",
	//     "a",
	//     99
	//   ],
	//   [
	//     " ",
	//     "baz",
	//     null,
	//     [
	//       [
	//         " ",
	//         "a",
	//         {
	//           "d": "apples-and-oranges"
	//         }
	//       ],
	//       [
	//         "+",
	//         "e",
	//         "thirty-thousand-something-dogecoin"
	//       ]
	//     ]
	//   ]
	// ]
}

type TestCase struct {
	description string // description of what test is checking
	src, dst    string // express test cases as json strings
	expect      Deltas // expected output
}

func RunTestCases(t *testing.T, cases []TestCase, opts ...DiffOption) {
	var (
		src    interface{}
		result interface{}
		dst    interface{}
		dd     = New(opts...)
		ctx    = context.Background()
	)

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			if err := json.Unmarshal([]byte(c.src), &src); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(c.src), &result); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(c.dst), &dst); err != nil {
				t.Fatal(err)
			}

			diff, err := dd.Diff(ctx, result, dst)
			if err != nil {
				t.Fatalf("Diff error: %s", err)
			}

			if diffDiff := cmp.Diff(c.expect, diff); diffDiff != "" {
				t.Errorf("diff script response mismatch (-want +got):\n%s", diffDiff)
			}

			if err := Patch(diff, &result); err != nil {
				t.Errorf("error patching source: %s", err)
			}

			if diff := cmp.Diff(dst, result); diff != "" {
				srcData, _ := json.Marshal(src)
				dstData, _ := json.Marshal(dst)
				t.Errorf("patched result mismatch:\nsrc  : %s\ndst  : %s\ndiff (-want, +got):\n%s\n", string(srcData), string(dstData), diff)
			}
		})
	}
}

func TestBasicDiffing(t *testing.T) {
	cases := []TestCase{
		{
			"scalar change array",
			`[[0,1,2]]`,
			`[[0,1,3]]`,
			Deltas{
				{Type: DTContext, Path: IndexAddr(0), Deltas: Deltas{
					{Type: DTContext, Path: IndexAddr(0), Value: float64(0)},
					{Type: DTContext, Path: IndexAddr(1), Value: float64(1)},
					{Type: DTDelete, Path: IndexAddr(2), Value: float64(2)},
					{Type: DTInsert, Path: IndexAddr(2), Value: float64(3)},
				}},
			},
		},
		{
			"scalar change object",
			`{"a":[0,1,2],"b":true}`,
			`{"a":[0,1,3],"b":true}`,
			Deltas{
				{Type: DTContext, Path: StringAddr("a"), Deltas: Deltas{
					{Type: DTContext, Path: IndexAddr(0), Value: float64(0)},
					{Type: DTContext, Path: IndexAddr(1), Value: float64(1)},
					{Type: DTDelete, Path: IndexAddr(2), Value: float64(2)},
					{Type: DTInsert, Path: IndexAddr(2), Value: float64(3)},
				}},
				{Type: DTContext, Path: StringAddr("b"), Value: true},
			},
		},
		{
			"insert into array",
			`[[1]]`,
			`[[1],[2]]`,
			Deltas{
				{Type: DTContext, Path: IndexAddr(0), Value: []interface{}{float64(1)}},
				{Type: DTInsert, Path: IndexAddr(1), Value: []interface{}{float64(2)}},
			},
		},
		{
			"insert into object",
			`{"a":[1]}`,
			`{"a":[1],"b":[2]}`,
			Deltas{
				{Type: DTContext, Path: StringAddr("a"), Value: []interface{}{float64(1)}},
				{Type: DTInsert, Path: StringAddr("b"), Value: []interface{}{float64(2)}},
			},
		},
		{
			"delete from array",
			`[[1],[2],[3]]`,
			`[[1],[3]]`,
			Deltas{
				{Type: DTContext, Path: IndexAddr(0), Value: []interface{}{float64(1)}},
				{Type: DTDelete, Path: IndexAddr(1), Value: []interface{}{float64(2)}},
				{Type: DTContext, Path: IndexAddr(1), Value: []interface{}{float64(3)}},
			},
		},
		{
			"delete from object",
			`{"a":[false],"b":[2],"c":[3]}`,
			`{"a":[false],"c":[3]}`,
			Deltas{
				{Type: DTContext, Path: StringAddr("a"), Value: []interface{}{false}},
				{Type: DTDelete, Path: StringAddr("b"), Value: []interface{}{float64(2)}},
				{Type: DTContext, Path: StringAddr("c"), Value: []interface{}{float64(3)}},
			},
		},
		{
			"key change case",
			`{"a":[1],"b":[2],"c":[3]}`,
			`{"A":[1],"b":[2],"c":[3]}`,
			Deltas{
				{Type: DTInsert, Path: StringAddr("A"), Value: []interface{}{float64(1)}},
				{Type: DTDelete, Path: StringAddr("a"), Value: []interface{}{float64(1)}},
				{Type: DTContext, Path: StringAddr("b"), Value: []interface{}{float64(2)}},
				{Type: DTContext, Path: StringAddr("c"), Value: []interface{}{float64(3)}},
			},
		},

		// TODO (b5) - another problematic diff case
		// {
		// 	"diff object and array children",
		// 	`{"a":[0,1,2], "b": true }`,
		// 	`{"a":{"foo": [0,1,2] }, "b": false }`,
		// 	Deltas{},
		// },

		// TODO (b5) - I think this should be an error. These inputs don't share a
		// common root data type
		// {
		// 	"object-to-array root",
		// 	`[ [1,2,3], [4,5,6], [7,8,9] ]`,
		// 	`{ "foo": [1,2,3], "baz": { "bat": [false]}}`,
		// 	Deltas{},
		// },
	}

	RunTestCases(t, cases)
}

func TestChangeDiffs(t *testing.T) {
	cases := []TestCase{
		{
			"scalar change array",
			`[[0,1,2]]`,
			`[[0,1,3]]`,
			Deltas{
				{Type: DTContext, Path: IndexAddr(0), Deltas: Deltas{
					{Type: DTContext, Path: IndexAddr(0), Value: float64(0)},
					{Type: DTContext, Path: IndexAddr(1), Value: float64(1)},
					{Type: DTUpdate, Path: IndexAddr(2), SourceValue: float64(2), Value: float64(3)},
				}},
			},
		},
		{
			"scalar change object",
			`{"a":[0,1,2],"b":true}`,
			`{"a":[0,1,3],"b":true}`,
			Deltas{
				{Type: DTContext, Path: StringAddr("a"), Deltas: Deltas{
					{Type: DTContext, Path: IndexAddr(0), Value: float64(0)},
					{Type: DTContext, Path: IndexAddr(1), Value: float64(1)},
					{Type: DTUpdate, Path: IndexAddr(2), SourceValue: float64(2), Value: float64(3)}},
				},
				{Type: DTContext, Path: StringAddr("b"), Value: true},
			},
		},
	}

	RunTestCases(t, cases, func(c *Config) { c.Changes = true })
}

func TestInsertGeneralizing(t *testing.T) {
	cases := []TestCase{
		{
			"grouping object insertion",
			`[{"a":"a", "b":"b"},{"c":"c"}]`,
			`[{"a":"a", "b":"b"},{"c":"c","d":{"this":"is","a":"big","insertion":{"object":5,"nesting":[true]}}}]`,
			Deltas{
				{Type: DTContext, Path: IndexAddr(0), Value: map[string]interface{}{"a": "a", "b": "b"}},
				{Type: DTContext, Path: IndexAddr(1), Deltas: Deltas{
					{Type: DTContext, Path: StringAddr("c"), Value: "c"},
					{Type: DTInsert, Path: StringAddr("d"), Value: map[string]interface{}{
						"this": "is",
						"a":    "big",
						"insertion": map[string]interface{}{
							"object":  float64(5),
							"nesting": []interface{}{true},
						},
					}},
				}},
			},
		},
		// {
		// 	"real-world large stats object insertion",
		// 	`{"bodyPath":"/ipfs/QmUNYnjzjTJyBEY3gXzQuGaXeawoFpmCi3UxjpbN4mvnib","commit":{"author":{"id":"QmSyDX5LYTiwQi861F5NAwdHrrnd1iRGsoEvCyzQMUyZ4W"},"path":"/ipfs/QmcHeeUmiDQE97rHw8GSCKWfsMXsLyqw1xrwxDA34XSqNE","qri":"cm:0","signature":"jq8TIriZaUqWyoXwr/vhPZyuZkxFttL9Bse67yoPszWPdKn8KhO7+DGBkVc/VQYdNaGoWRLajRtlcv8avp5RADyJEA3hc2SGsfYW4X+I5Wyj6ckD9p4UfRMrYakJT5yGDlfa0OW0T306k6VTt3v4O93Jj1hBNS45xsZ/TKSRGwiA9l5uh2Xt2XMTRPeFvDImdTomhB5mZBfLCHp7tj2i7G892JQPz9lidiyq0KrF7I6xRXbCoW3DMq9q63xWCnN8dnUpOEn+mupv+KL36Dzl3cE78fcKL0M/6WHP9T4OxyaQ/CEYOQA4RlJbcXMX9jLFnYsCht8Vxq7ffqTlRKP8lA==","timestamp":"2019-02-22T14:21:27.038532Z","title":"created dataset"},"meta":{"accessPath":"https://theunitedstates.io/","citations":[{}],"description":"Processed list of current legislators from @unitedstates.\n\n@unitedstates is a shared commons of data and tools for the United States. Made by the public, used by the public. ","downloadPath":"https://theunitedstates.io/congress-legislators/legislators-current.json","keywords":["us","gov","congress","538"],"license":{"type":"CC0 - Creative Commons Zero Public Domain Dedication","url":"https://creativecommons.org/publicdomain/zero/1.0/"},"qri":"md:0","theme":["government"],"title":"US Members of Congress"},"name":"us_current_legislators","path":"/ipfs/QmST56YbcS7Um3vpwLwWDTZozPHPttc1N6Jd19uU1W2z4t/dataset.json","peername":"b5","qri":"ds:0","structure":{"checksum":"QmXzzSj4UNqdCo4yX3t6ELfFi5QoEyj8zi9mkqiJofN1PC","depth":2,"errCount":0,"entries":538,"format":"json","length":87453,"qri":"st:0","schema":{"type":"array"}},"transform":{"qri":"tf:0","scriptPath":"/ipfs/QmSzYwaciz5C75BGzqVho24ngmhwMm5CcqVUPrPAwqPNWc","syntax":"starlark","syntaxVersion":"0.2.2-dev"}}`,
		// 	`{"bodyPath":"/ipfs/QmUNYnjzjTJyBEY3gXzQuGaXeawoFpmCi3UxjpbN4mvnib","commit":{"author":{"id":"QmSyDX5LYTiwQi861F5NAwdHrrnd1iRGsoEvCyzQMUyZ4W"},"path":"/ipfs/QmR5JTQxxjJPrZBL4neynAyv2WLuXQujR9NoLkfcahc34W","qri":"cm:0","signature":"jy3JiFNVgcGn8pcm1Vuv9Z3AbVl18Yh7z3Bj+N8t5lz0/OY+ZxbBrNPXVx/M6FgbPA9RzFGzgJ8xKudBsqS94kJaQ9yg2zvNmZxufiFs3YxoIhxPabod0fY5Whq91Ns3Ov3AOCKarIYpXyAdFDvpRQ3VSyqwaTNc9lheutEDeFHmW5BGFNsA/NXhbPIocgE3G48PYUXIRInwaFhsLjpcFSwn/cG+Xbkly0OrOYtCTS5hZ0aBPbk6FAAu6l6BVGbxDduflYyt8UFpdiinJf8S7G+l5nwO0VlQwTT47q3CkcPAdQTtTxHnz4mYwaWPGeqryBi4TO6PXlmbRDLaQ8v3dQ==","timestamp":"2019-02-23T23:12:25.886874Z","title":"forced update"},"meta":{"accessPath":"https://theunitedstates.io/","citations":[{}],"description":"Processed list of current legislators from @unitedstates.\n\n@unitedstates is a shared commons of data and tools for the United States. Made by the public, used by the public. ","downloadPath":"https://theunitedstates.io/congress-legislators/legislators-current.json","keywords":["us","gov","congress","538"],"license":{"type":"CC0 - Creative Commons Zero Public Domain Dedication","url":"https://creativecommons.org/publicdomain/zero/1.0/"},"qri":"md:0","theme":["government"],"title":"US Members of Congress"},"name":"us_current_legislators","path":"/ipfs/QmTV1n5BfQnG4EigyRJUP3466FRPgDFEbckva6mEmtLR97/dataset.json","peername":"b5","previousPath":"/ipfs/QmST56YbcS7Um3vpwLwWDTZozPHPttc1N6Jd19uU1W2z4t","qri":"ds:0","stats":{"bioguide":{"count":538,"maxLength":7,"minLength":7,"type":"string"},"birthday":{"count":538,"maxLength":10,"minLength":10,"type":"string"},"first":{"count":538,"maxLength":11,"minLength":2,"type":"string"},"full":{"count":538,"maxLength":30,"minLength":6,"type":"string"},"gender":{"count":538,"maxLength":1,"minLength":1,"type":"string"},"last":{"count":538,"maxLength":17,"minLength":3,"type":"string"},"party":{"count":538,"maxLength":11,"minLength":8,"type":"string"},"religion":{"count":538,"max":0,"min":0,"type":"integer"},"state":{"count":538,"maxLength":2,"minLength":2,"type":"string"}},"structure":{"checksum":"QmXzzSj4UNqdCo4yX3t6ELfFi5QoEyj8zi9mkqiJofN1PC","depth":2,"errCount":0,"entries":538,"format":"json","length":87453,"qri":"st:0","schema":{"type":"array"}},"transform":{"qri":"tf:0","scriptPath":"/ipfs/QmSzYwaciz5C75BGzqVho24ngmhwMm5CcqVUPrPAwqPNWc","syntax":"starlark","syntaxVersion":"0.2.2-dev"}}`,
		// 	Deltas{
		// 		{Type: DTDelete, Path: "/commit/path", Value: "/ipfs/QmcHeeUmiDQE97rHw8GSCKWfsMXsLyqw1xrwxDA34XSqNE"},
		// 		{Type: DTInsert, Path: "/commit/path", Value: "/ipfs/QmR5JTQxxjJPrZBL4neynAyv2WLuXQujR9NoLkfcahc34W"},
		// 		{Type: DTDelete, Path: "/commit/signature", Value: "jq8TIriZaUqWyoXwr/vhPZyuZkxFttL9Bse67yoPszWPdKn8KhO7+DGBkVc/VQYdNaGoWRLajRtlcv8avp5RADyJEA3hc2SGsfYW4X+I5Wyj6ckD9p4UfRMrYakJT5yGDlfa0OW0T306k6VTt3v4O93Jj1hBNS45xsZ/TKSRGwiA9l5uh2Xt2XMTRPeFvDImdTomhB5mZBfLCHp7tj2i7G892JQPz9lidiyq0KrF7I6xRXbCoW3DMq9q63xWCnN8dnUpOEn+mupv+KL36Dzl3cE78fcKL0M/6WHP9T4OxyaQ/CEYOQA4RlJbcXMX9jLFnYsCht8Vxq7ffqTlRKP8lA=="},
		// 		{Type: DTInsert, Path: "/commit/signature", Value: "jy3JiFNVgcGn8pcm1Vuv9Z3AbVl18Yh7z3Bj+N8t5lz0/OY+ZxbBrNPXVx/M6FgbPA9RzFGzgJ8xKudBsqS94kJaQ9yg2zvNmZxufiFs3YxoIhxPabod0fY5Whq91Ns3Ov3AOCKarIYpXyAdFDvpRQ3VSyqwaTNc9lheutEDeFHmW5BGFNsA/NXhbPIocgE3G48PYUXIRInwaFhsLjpcFSwn/cG+Xbkly0OrOYtCTS5hZ0aBPbk6FAAu6l6BVGbxDduflYyt8UFpdiinJf8S7G+l5nwO0VlQwTT47q3CkcPAdQTtTxHnz4mYwaWPGeqryBi4TO6PXlmbRDLaQ8v3dQ=="},
		// 		{Type: DTDelete, Path: "/commit/timestamp", Value: "2019-02-22T14:21:27.038532Z"},
		// 		{Type: DTInsert, Path: "/commit/timestamp", Value: "2019-02-23T23:12:25.886874Z"},
		// 		{Type: DTDelete, Path: "/commit/title", Value: "created dataset"},
		// 		{Type: DTInsert, Path: "/commit/title", Value: "forced update"},
		// 		{Type: DTDelete, Path: "/path", Value: "/ipfs/QmST56YbcS7Um3vpwLwWDTZozPHPttc1N6Jd19uU1W2z4t/dataset.json"},
		// 		{Type: DTInsert, Path: "/path", Value: "/ipfs/QmTV1n5BfQnG4EigyRJUP3466FRPgDFEbckva6mEmtLR97/dataset.json"},
		// 		{Type: DTInsert, Path: "/previousPath", Value: "/ipfs/QmST56YbcS7Um3vpwLwWDTZozPHPttc1N6Jd19uU1W2z4t"},
		// 		{Type: DTInsert, Path: "/stats", Value: map[string]interface{}{
		// 			"bioguide": map[string]interface{}{"count": float64(538), "maxLength": float64(7), "minLength": float64(7), "type": "string"},
		// 			"birthday": map[string]interface{}{"count": float64(538), "maxLength": float64(10), "minLength": float64(10), "type": "string"},
		// 			"first":    map[string]interface{}{"count": float64(538), "maxLength": float64(11), "minLength": float64(2), "type": "string"},
		// 			"full":     map[string]interface{}{"count": float64(538), "maxLength": float64(30), "minLength": float64(6), "type": "string"},
		// 			"gender":   map[string]interface{}{"count": float64(538), "maxLength": float64(1), "minLength": float64(1), "type": "string"},
		// 			"last":     map[string]interface{}{"count": float64(538), "maxLength": float64(17), "minLength": float64(3), "type": "string"},
		// 			"party":    map[string]interface{}{"count": float64(538), "maxLength": float64(11), "minLength": float64(8), "type": "string"},
		// 			"religion": map[string]interface{}{"count": float64(538), "max": float64(0), "min": float64(0), "type": "integer"},
		// 			"state":    map[string]interface{}{"count": float64(538), "maxLength": float64(2), "minLength": float64(2), "type": "string"},
		// 		},
		// 		},
		// 	},
		// },
	}

	RunTestCases(t, cases)
}

func TestRootChanges(t *testing.T) {
	t.Skip("TODO (b5) - fix this with 'null' as the position indicator for root object changes")

	cases := []TestCase{
		{
			"large structure overflow",
			`{ "qri": "ds:0" }`,
			`[ "ds:0", ["rank","probability_of_automation","soc_code","job_title"] ]`,
			Deltas{},
		},
	}

	RunTestCases(t, cases)

	var aJSON = `{ "qri": "ds:0" }`
	var bJSON = `[ "ds:0", ["rank","probability_of_automation","soc_code","job_title"] ]`

	var a interface{}
	if err := json.Unmarshal([]byte(aJSON), &a); err != nil {
		panic(err)
	}

	var b interface{}
	if err := json.Unmarshal([]byte(bJSON), &b); err != nil {
		panic(err)
	}

	d := &diff{d1: a, d2: b}
	d.t1, d.t2, d.t1Nodes = d.prepTrees(context.Background())
	d.queueMatch(d.t1Nodes, d.t2)
	d.optimize(d.t1, d.t2)

	buf := dotGraphTree(d)
	ioutil.WriteFile("testdata/graph_2.dot", buf.Bytes(), os.ModePerm)

	delts := d.calcDeltas(d.t1, d.t2)
	deltas, _ := json.MarshalIndent(delts, "  ", "")
	t.Log(string(deltas))
}

func TestDiffDotGraph(t *testing.T) {
	var aJSON = `{
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
	}`

	var bJSON = `{
		"a": 99,
		"foo": [1,2,3],
		"bar": false,
		"baz": {
			"a": {
				"b": 5,
				"c": false,
				"d": "apples-and-oranges"
			},
			"e": "thirty-thousand-something-dollars",
			"f": false
		}
	}`

	var a interface{}
	if err := json.Unmarshal([]byte(aJSON), &a); err != nil {
		panic(err)
	}

	var b interface{}
	if err := json.Unmarshal([]byte(bJSON), &b); err != nil {
		panic(err)
	}

	d := &diff{d1: a, d2: b}
	d.t1, d.t2, d.t1Nodes = d.prepTrees(context.Background())
	d.queueMatch(d.t1Nodes, d.t2)
	d.optimize(d.t1, d.t2)

	buf := dotGraphTree(d)
	ioutil.WriteFile("testdata/graph.dot", buf.Bytes(), os.ModePerm)
}

func pathString(addrs []Addr) string {
	p := ""
	for _, addr := range addrs {
		p += "/" + addr.String()
	}
	return p
}

func dotGraphTree(d *diff) *bytes.Buffer {
	mkID := func(pfx string, n node) string {
		id := strings.Replace(pathString(path(n)), "/", "", -1)
		if id == pfx {
			id = "root"
		}
		return pfx + id
	}

	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "digraph {\n")

	fmt.Fprintf(buf, "  subgraph cluster_t1 {\n")
	fmt.Fprintf(buf, "    label=\"t1\";\n")

	walk(d.t1, nil, func(p []Addr, n node) bool {
		if cmp, ok := n.(compound); ok {
			pID := mkID("t1", cmp)
			fmt.Fprintf(buf, "    %s [label=\"%s\", tooltip=\"weight: %d\"];\n", pID, pathString(p), n.Weight())
			for _, ch := range cmp.Children() {
				fmt.Fprintf(buf, "    %s -> %s;\n", pID, mkID("t1", ch))
			}
		}
		return true
	})
	fmt.Fprintf(buf, "  }\n")

	fmt.Fprintf(buf, "  subgraph cluster_t2 {\n")
	fmt.Fprintf(buf, "    label=\"t2\";\n")
	walk(d.t2, nil, func(p []Addr, n node) bool {
		if cmp, ok := n.(compound); ok {
			pID := mkID("t2", cmp)
			fmt.Fprintf(buf, "    %s [label=\"%s\", tooltip=\"weight: %d\"];\n", pID, pathString(p), n.Weight())
			for _, ch := range cmp.Children() {
				fmt.Fprintf(buf, "    %s -> %s;\n", pID, mkID("t2", ch))
			}
		}
		return true
	})
	fmt.Fprintf(buf, "  }\n\n")

	walk(d.t2, nil, func(p []Addr, n node) bool {
		nID := mkID("t2", n)
		if n.Match() != nil {
			fmt.Fprintf(buf, "  %s -> %s[color=red,penwidth=1.0];\n", nID, mkID("t1", n.Match()))
		}
		return true
	})

	fmt.Fprintf(buf, "}")
	return buf
}

// most of our test suite uses json marshaling for convenience. The json package
// only works with float64s. This confirms ints work as well
func TestDiffIntData(t *testing.T) {
	leftData := []interface{}{
		[]interface{}{int64(1), int64(2), int64(3)},
		[]interface{}{int64(4), int64(5), int64(6)},
		[]interface{}{int64(7), int64(8), int64(9)},
	}
	rightData := []interface{}{
		[]interface{}{int64(1), int64(2), int64(3)},
		[]interface{}{int64(4), int64(0), int64(6)},
		[]interface{}{int64(10), int64(8), int64(9)},
	}

	diff, err := New().Diff(context.Background(), leftData, rightData)
	if err != nil {
		t.Fatalf("Diff error: %s", err)
	}

	expect := Deltas{
		{Type: DTContext, Path: IndexAddr(0), Value: []interface{}{int64(1), int64(2), int64(3)}},
		{Type: DTContext, Path: IndexAddr(1), Deltas: Deltas{
			{Type: DTContext, Path: IndexAddr(0), Value: int64(4)},
			{Type: DTDelete, Path: IndexAddr(1), Value: int64(5)},
			{Type: DTInsert, Path: IndexAddr(1), Value: int64(0)},
			{Type: DTContext, Path: IndexAddr(2), Value: int64(6)},
		}},
		{Type: DTContext, Path: IndexAddr(2), Deltas: Deltas{
			{Type: DTDelete, Path: IndexAddr(0), Value: int64(7)},
			{Type: DTInsert, Path: IndexAddr(0), Value: int64(10)},
			{Type: DTContext, Path: IndexAddr(1), Value: int64(8)},
			{Type: DTContext, Path: IndexAddr(2), Value: int64(9)},
		}},
	}

	if diffDiff := cmp.Diff(expect, diff); diffDiff != "" {
		t.Errorf("delta mismatch. (-want +got):\n%s", diffDiff)
	}
}

func TestDiffStats(t *testing.T) {
	leftData := map[string]interface{}{
		"a": "apple",
		"b": []interface{}{
			[]interface{}{"one", "two", "three"},
			[]interface{}{"four", "five", "six"},
		},
	}
	rightData := map[string]interface{}{
		"a": "apple",
		"b": []interface{}{},
	}

	diff, stat, err := New().StatDiff(context.Background(), leftData, rightData)
	if err != nil {
		t.Fatalf("Diff error: %s", err)
	}

	expect := Deltas{
		{Type: DTContext, Path: StringAddr("a"), Value: "apple"},
		{Type: DTContext, Path: StringAddr("b"), Deltas: Deltas{
			{Type: DTDelete, Path: IndexAddr(0), Value: []interface{}{"one", "two", "three"}},
			{Type: DTDelete, Path: IndexAddr(0), Value: []interface{}{"four", "five", "six"}},
		}},
	}
	if diffDiff := cmp.Diff(expect, diff); diffDiff != "" {
		t.Errorf("deltas mismatch (-want +got):\n%s", diffDiff)
	}

	expectStat := &Stats{
		Left:        11,
		Right:       3,
		LeftWeight:  107,
		RightWeight: 13,
		Deletes:     2,
	}
	if diff := cmp.Diff(expectStat, stat); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}
}

// func BenchmarkDiff1(b *testing.B) {
// 	srcData := `{
// 		"foo" : {
// 			"bar" : [1,2,3]
// 		},
// 		"baz" : [4,5,6],
// 		"bat" : false
// 	}`

// 	dstData := `{
// 		"baz" : [7,8,9],
// 		"bat" : true,
// 		"champ" : {
// 			"bar" : [1,2,3]
// 		}
// 	}`

// 	var (
// 		src, dst interface{}
// 		ctx      = context.Background()
// 		dd       = New()
// 	)
// 	if err := json.Unmarshal([]byte(srcData), &src); err != nil {
// 		b.Fatal(err)
// 	}
// 	if err := json.Unmarshal([]byte(dstData), &dst); err != nil {
// 		b.Fatal(err)
// 	}

// 	for n := 0; n < b.N; n++ {
// 		dd.Diff(ctx, src, dst)
// 	}
// }

// func BenchmarkDiffDatasets(b *testing.B) {
// 	var (
// 		diff  = New()
// 		data1 = []byte(`{"body":[["a","b","c","d"],["1","2","3","4"],["e","f","g","h"]],"bodyPath":"/ipfs/QmP2tdkqc4RhSDGv1KSWoJw1pwzNu6HzMcYZaVFkLN9PMc","commit":{"author":{"id":"QmSyDX5LYTiwQi861F5NAwdHrrnd1iRGsoEvCyzQMUyZ4W"},"path":"/ipfs/QmbwJNx88xNknXYewLCVBVJqbZ5oaiffr4WYDoCJAuCZ93","qri":"cm:0","signature":"TUREFCfoKEf5J189c0jdKfleRYsGZm8Q6sm6g6lJctXGDDM8BGdpSVjMltGTmmrtN6qtQJKRail5ceG325Rb8hLYoMe4926gXZNWBlMfD0yBHSjo81LsE25UqVeloU2W19Z1MNOrLTDPDRBoM0g3vyJLykGQ0UPRqpUvXNod0E5ONZOKGrQpByp113h12yiAjsiCBR6sAfIScNpcyjzkiDhBCCbMy9cGfMVK8q7wNCmcC41zguGhvv1biDoE+MEVDc1QPN1dYeEaDsvaRu5jWSv44zhVdC3lZtlT8R9qArk8OQVW798ctQ6NJ5kCiZ3C6Z19VPrptr85oknoNNaYxA==","timestamp":"2019-02-04T14:26:43.158109Z","title":"created dataset"},"name":"test_1","path":"/ipfs/QmeSYBYd3LVsFPRp1jiXgT8q22Md3R7swUzd9yt7MPVUcj/dataset.json","peername":"b5","qri":"ds:0","structure":{"depth":2,"errCount":0,"format":"json","qri":"st:0","schema":{"type":"array"}}}`)
// 		data2 = []byte(`{"body":[["a","b","c","d"],["1","2","3","4"],["e","f","g","h"]],"bodyPath":"/ipfs/QmP2tdkqc4RhSDGv1KSWoJw1pwzNu6HzMcYZaVFkLN9PMc","commit":{"author":{"id":"QmSyDX5LYTiwQi861F5NAwdHrrnd1iRGsoEvCyzQMUyZ4W"},"path":"/ipfs/QmVZrXZ2d6DF11BL7QLJ8AYFYaNiLgAWVEshZ3HB5ogZJS","qri":"cm:0","signature":"CppvSyFkaLNIY3lIOGxq7ybA18ZzJbgrF7XrIgrxi7pwKB3RGjriaCqaqTGNMTkdJCATN/qs/Yq4IIbpHlapIiwfzVHFUO8m0a2+wW0DHI+y1HYsRvhg3+LFIGHtm4M+hqcDZg9EbNk8weZI+Q+FPKk6VjPKpGtO+JHV+nEFovFPjS4XMMoyuJ96KiAEeZISuF4dN2CDSV+WC93sMhdPPAQJJZjZX+3cc/fOaghOkuhedXaA0poTVJQ05aAp94DyljEnysuS7I+jfNrsE/6XhtazZnOSYX7e0r1PJwD7OdoZYRH73HnDk+Q9wg6RrpU7EehF39o4UywyNGAI5yJkxg==","timestamp":"2019-02-11T17:50:20.501283Z","title":"forced update"},"name":"test_1","path":"/ipfs/QmaAuKZezio5knAFXU4krPcZfBWHnHDWWKEX32Ne9v6niQ/dataset.json","peername":"b5","previousPath":"/ipfs/QmeSYBYd3LVsFPRp1jiXgT8q22Md3R7swUzd9yt7MPVUcj","qri":"ds:0","structure":{"depth":2,"errCount":0,"format":"json","qri":"st:0","schema":{"type":"array"}}}`)
// 		t1    interface{}
// 		t2    interface{}
// 		ctx   = context.Background()
// 	)
// 	if err := json.Unmarshal(data1, &t1); err != nil {
// 		b.Fatal(err)
// 	}
// 	if err := json.Unmarshal(data2, &t2); err != nil {
// 		b.Fatal(err)
// 	}
// 	for i := 0; i < b.N; i++ {
// 		diff.Diff(ctx, t1, t2)
// 	}
// }

// func BenchmarkDiff5MB(b *testing.B) {
// 	diff := New()
// 	ctx := context.Background()

// 	f1, err := os.Open("testdata/airport_codes.json")
// 	if err != nil {
// 		b.Fatal(err)
// 	}
// 	var t1 map[string]interface{}
// 	if err := json.NewDecoder(f1).Decode(&t1); err != nil {
// 		b.Fatal(err)
// 	}
// 	f2, err := os.Open("testdata/airport_codes_2.json")
// 	if err != nil {
// 		b.Fatal(err)
// 	}
// 	var t2 map[string]interface{}
// 	if err := json.NewDecoder(f2).Decode(&t2); err != nil {
// 		b.Fatal(err)
// 	}
// 	for i := 0; i < b.N; i++ {
// 		diff.Diff(ctx, t1, t2)
// 	}
// }
