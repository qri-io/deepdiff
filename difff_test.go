package difff

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"
)

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

func TestDiffDotGraph(t *testing.T) {
	var a interface{}
	if err := json.Unmarshal([]byte(aJSON), &a); err != nil {
		panic(err)
	}

	var b interface{}
	if err := json.Unmarshal([]byte(bJSON), &b); err != nil {
		panic(err)
	}

	d := &diff{cfg: &DiffConfig{}, d1: a, d2: b}
	d.t1, d.t2, d.t1Nodes = d.prepTrees()
	d.queueMatch(d.t1Nodes, d.t2)
	d.optimize(d.t1, d.t2)

	mkID := func(pfx string, n Node) string {
		id := strings.Replace(path(n), "/", "", -1)
		if id == pfx {
			id = "root"
		}
		return pfx + id
	}

	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "digraph {\n")

	fmt.Fprintf(buf, "  subgraph cluster_t1 {\n")
	fmt.Fprintf(buf, "    label=\"t1\";\n")

	walk(d.t1, "t1", func(p string, n Node) bool {
		if cmp, ok := n.(Compound); ok {
			pID := mkID("t1", cmp)
			fmt.Fprintf(buf, "    %s [label=\"%s\", tooltip=\"weight: %d\"];\n", pID, p, n.Weight())
			for _, ch := range cmp.Children() {
				fmt.Fprintf(buf, "    %s -> %s;\n", pID, mkID("t1", ch))
			}
		}
		return true
	})
	fmt.Fprintf(buf, "  }\n")

	fmt.Fprintf(buf, "  subgraph cluster_t2 {\n")
	fmt.Fprintf(buf, "    label=\"t2\";\n")
	walk(d.t2, "t2", func(p string, n Node) bool {
		if cmp, ok := n.(Compound); ok {
			pID := mkID("t2", cmp)
			fmt.Fprintf(buf, "    %s [label=\"%s\", tooltip=\"weight: %d\"];\n", pID, p, n.Weight())
			for _, ch := range cmp.Children() {
				fmt.Fprintf(buf, "    %s -> %s;\n", pID, mkID("t2", ch))
			}
		}
		return true
	})
	fmt.Fprintf(buf, "  }\n\n")

	walk(d.t2, "", func(p string, n Node) bool {
		nID := mkID("t2", n)
		if n.Match() != nil {
			fmt.Fprintf(buf, "  %s -> %s[color=red,penwidth=1.0];\n", nID, mkID("t1", n.Match()))
		}
		return true
	})

	fmt.Fprintf(buf, "}")

	ioutil.WriteFile("testdata/graph.dot", buf.Bytes(), os.ModePerm)
}

// func TestBasicDiff(t *testing.T) {
// 	var a interface{}
// 	if err := json.Unmarshal([]byte(aJSON), &a); err != nil {
// 		panic(err)
// 	}

// 	var b interface{}
// 	if err := json.Unmarshal([]byte(bJSON), &b); err != nil {
// 		panic(err)
// 	}

// 	// TODO (b5): test output
// 	// Diff(a, b)
// 	_, := Diff(a, b)
// 	// data, _ := json.MarshalIndent(ds, "", "  ")
// 	// fmt.Println(string(data))
// }

type TestCase struct {
	description string   // description of what test is checking
	src, dst    string   // express test cases as json strings
	expect      []*Delta // expected output
}

func RunTestCases(t *testing.T, cases []TestCase, opts ...DiffOption) {
	var (
		src interface{}
		dst interface{}
	)

	for i, c := range cases {
		if err := json.Unmarshal([]byte(c.src), &src); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(c.dst), &dst); err != nil {
			t.Fatal(err)
		}

		diff := Diff(src, dst, opts...)
		if err := CompareDiffs(c.expect, diff); err != nil {
			t.Errorf("%d. '%s' result mismatch: %s", i, c.description, err)
		}
	}
}

func CompareDiffs(a, b []*Delta) error {
	if len(a) != len(b) {
		ad, _ := json.MarshalIndent(a, "", " ")
		bd, _ := json.MarshalIndent(b, "", " ")
		return fmt.Errorf("length mismatch: %d != %d\na: %v\nb: %v", len(a), len(b), string(ad), string(bd))
	}

	for i, delt := range a {
		if err := CompareDeltas(delt, b[i]); err != nil {
			return fmt.Errorf("%d: %s", i, err)
		}
	}

	return nil
}

func CompareDeltas(a, b *Delta) error {
	if a.Type != b.Type {
		return fmt.Errorf("Type mismatch. %s != %s", a.Type, b.Type)
	}

	if a.SrcPath != b.SrcPath {
		return fmt.Errorf("SrcPath mismatch. %s != %s", a.SrcPath, b.SrcPath)
	}

	if a.DstPath != b.DstPath {
		return fmt.Errorf("DstPath mismatch. %s != %s", a.DstPath, b.DstPath)
	}

	if !reflect.DeepEqual(a.SrcVal, b.SrcVal) {
		return fmt.Errorf("SrcVal mismatch. %v (%T) != %v (%T)", a.SrcVal, a.SrcVal, b.SrcVal, b.SrcVal)
	}
	if !reflect.DeepEqual(a.DstVal, b.DstVal) {
		return fmt.Errorf("DstVal mismatch. %v != %v", a.DstVal, b.DstVal)
	}

	return nil
}

func TestBasicDiffing(t *testing.T) {
	cases := []TestCase{
		{
			"scalar change array",
			`[[0,1,2]]`,
			`[[0,1,3]]`,
			[]*Delta{
				{Type: DTUpdate, SrcPath: "/0/2", DstPath: "/0/2", SrcVal: float64(2), DstVal: float64(3)},
			},
		},
		{
			"scalar change object",
			`{"a":[0,1,2]}`,
			`{"a":[0,1,3]}`,
			[]*Delta{
				{Type: DTUpdate, SrcPath: "/a/2", DstPath: "/a/2", SrcVal: float64(2), DstVal: float64(3)},
			},
		},
		{
			"insert array",
			`[[1]]`,
			`[[1],[2]]`,
			[]*Delta{
				// TODO (b5): Need to decide on what expected insert path for arrays is. should it be the index
				// to *begin* insertion at (aka the index just before what will be the index of the new insertion)?
				{Type: DTInsert, SrcPath: "", DstPath: "/1", SrcVal: nil, DstVal: []interface{}{float64(2)}},
			},
		},
		{
			"insert object",
			`{"a":[1]}`,
			`{"a":[1],"b":[2]}`,
			[]*Delta{
				// TODO (b5): Need to decide on what expected insert path for arrays is. should it be the index
				// to *begin* insertion at (aka the index just before what will be the index of the new insertion)?
				{Type: DTInsert, SrcPath: "", DstPath: "/b", SrcVal: nil, DstVal: []interface{}{float64(2)}},
			},
		},
		{
			"delete array",
			`[[1],[2],[3]]`,
			`[[1],[3]]`,
			[]*Delta{
				{Type: DTDelete, SrcPath: "/1", DstPath: "", SrcVal: []interface{}{float64(2)}, DstVal: nil},
			},
		},
		{
			"delete object",
			`{"a":[1],"b":[2],"c":[3]}`,
			`{"a":[1],"c":[3]}`,
			[]*Delta{
				{Type: DTDelete, SrcPath: "/b", DstPath: "", SrcVal: []interface{}{float64(2)}, DstVal: nil},
			},
		},
	}

	RunTestCases(t, cases)
}

func TestMoveDiffs(t *testing.T) {
	cases := []TestCase{
		{
			"different parent move array",
			`[[1],[2],[3]]`,
			`[[1],[2,[3]]]`,
			[]*Delta{
				{Type: DTMove, SrcPath: "/2", DstPath: "/1/1", SrcVal: []interface{}{float64(3)}, DstVal: []interface{}{float64(3)}},
			},
		},
		{
			"same parent move array",
			`[[1],[2],[3]]`,
			`[[1],[3],[2]]`,
			[]*Delta{
				{Type: DTMove, SrcPath: "/2", DstPath: "/1", DstVal: []interface{}{float64(3)}},
			},
		},
	}
	RunTestCases(t, cases, func(o *DiffConfig) {
		o.MoveDeltas = true
	})
}

func BenchmarkDiff1(b *testing.B) {
	srcData := `{
		"foo" : {
			"bar" : [1,2,3]
		},
		"baz" : [4,5,6],
		"bat" : false
	}`

	dstData := `{
		"baz" : [7,8,9],
		"bat" : true,
		"champ" : {
			"bar" : [1,2,3]
		}
	}`

	var (
		src, dst interface{}
	)
	if err := json.Unmarshal([]byte(srcData), &src); err != nil {
		b.Fatal(err)
	}
	if err := json.Unmarshal([]byte(dstData), &dst); err != nil {
		b.Fatal(err)
	}

	for n := 0; n < b.N; n++ {
		Diff(src, dst)
	}
}

func BenchmarkDiffDatasets(b *testing.B) {
	var (
		data1 = []byte(`{"body":[["a","b","c","d"],["1","2","3","4"],["e","f","g","h"]],"bodyPath":"/ipfs/QmP2tdkqc4RhSDGv1KSWoJw1pwzNu6HzMcYZaVFkLN9PMc","commit":{"author":{"id":"QmSyDX5LYTiwQi861F5NAwdHrrnd1iRGsoEvCyzQMUyZ4W"},"path":"/ipfs/QmbwJNx88xNknXYewLCVBVJqbZ5oaiffr4WYDoCJAuCZ93","qri":"cm:0","signature":"TUREFCfoKEf5J189c0jdKfleRYsGZm8Q6sm6g6lJctXGDDM8BGdpSVjMltGTmmrtN6qtQJKRail5ceG325Rb8hLYoMe4926gXZNWBlMfD0yBHSjo81LsE25UqVeloU2W19Z1MNOrLTDPDRBoM0g3vyJLykGQ0UPRqpUvXNod0E5ONZOKGrQpByp113h12yiAjsiCBR6sAfIScNpcyjzkiDhBCCbMy9cGfMVK8q7wNCmcC41zguGhvv1biDoE+MEVDc1QPN1dYeEaDsvaRu5jWSv44zhVdC3lZtlT8R9qArk8OQVW798ctQ6NJ5kCiZ3C6Z19VPrptr85oknoNNaYxA==","timestamp":"2019-02-04T14:26:43.158109Z","title":"created dataset"},"name":"test_1","path":"/ipfs/QmeSYBYd3LVsFPRp1jiXgT8q22Md3R7swUzd9yt7MPVUcj/dataset.json","peername":"b5","qri":"ds:0","structure":{"depth":2,"errCount":0,"format":"json","qri":"st:0","schema":{"type":"array"}}}`)
		data2 = []byte(`{"body":[["a","b","c","d"],["1","2","3","4"],["e","f","g","h"]],"bodyPath":"/ipfs/QmP2tdkqc4RhSDGv1KSWoJw1pwzNu6HzMcYZaVFkLN9PMc","commit":{"author":{"id":"QmSyDX5LYTiwQi861F5NAwdHrrnd1iRGsoEvCyzQMUyZ4W"},"path":"/ipfs/QmVZrXZ2d6DF11BL7QLJ8AYFYaNiLgAWVEshZ3HB5ogZJS","qri":"cm:0","signature":"CppvSyFkaLNIY3lIOGxq7ybA18ZzJbgrF7XrIgrxi7pwKB3RGjriaCqaqTGNMTkdJCATN/qs/Yq4IIbpHlapIiwfzVHFUO8m0a2+wW0DHI+y1HYsRvhg3+LFIGHtm4M+hqcDZg9EbNk8weZI+Q+FPKk6VjPKpGtO+JHV+nEFovFPjS4XMMoyuJ96KiAEeZISuF4dN2CDSV+WC93sMhdPPAQJJZjZX+3cc/fOaghOkuhedXaA0poTVJQ05aAp94DyljEnysuS7I+jfNrsE/6XhtazZnOSYX7e0r1PJwD7OdoZYRH73HnDk+Q9wg6RrpU7EehF39o4UywyNGAI5yJkxg==","timestamp":"2019-02-11T17:50:20.501283Z","title":"forced update"},"name":"test_1","path":"/ipfs/QmaAuKZezio5knAFXU4krPcZfBWHnHDWWKEX32Ne9v6niQ/dataset.json","peername":"b5","previousPath":"/ipfs/QmeSYBYd3LVsFPRp1jiXgT8q22Md3R7swUzd9yt7MPVUcj","qri":"ds:0","structure":{"depth":2,"errCount":0,"format":"json","qri":"st:0","schema":{"type":"array"}}}`)
		t1    interface{}
		t2    interface{}
	)
	if err := json.Unmarshal(data1, &t1); err != nil {
		b.Fatal(err)
	}
	if err := json.Unmarshal(data2, &t2); err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		Diff(t1, t2)
	}
}

func BenchmarkDiff5MB(b *testing.B) {
	f1, err := os.Open("testdata/airport_codes.json")
	if err != nil {
		b.Fatal(err)
	}
	var t1 map[string]interface{}
	if err := json.NewDecoder(f1).Decode(&t1); err != nil {
		b.Fatal(err)
	}
	f2, err := os.Open("testdata/airport_codes_2.json")
	if err != nil {
		b.Fatal(err)
	}
	var t2 map[string]interface{}
	if err := json.NewDecoder(f2).Decode(&t2); err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		Diff(t1, t2)
	}
}
