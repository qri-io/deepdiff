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

	t1, t2, t1Nodes := prepTrees(a, b)
	queueMatch(t1Nodes, t2)

	mkID := func(pfx string, n Node) string {
		id := strings.Replace(path(n), ".", "", -1)
		if id == pfx {
			id = "root"
		}
		return pfx + id
	}

	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "digraph {\n")

	fmt.Fprintf(buf, "  subgraph t1 {\n")

	walk(t1, "", func(p string, n Node) {
		if cmp, ok := n.(Compound); ok {
			pID := mkID("t1", cmp)
			fmt.Fprintf(buf, "    %s [label=\"%s\", tooltip=\"weight: %d\"];\n", pID, p, n.Weight())
			for _, ch := range cmp.Children() {
				fmt.Fprintf(buf, "    %s -> %s;\n", pID, mkID("t1", ch))
			}
		}
	})
	fmt.Fprintf(buf, "  }\n")

	fmt.Fprintf(buf, "  subgraph t2 {\n")
	walk(t2, "", func(p string, n Node) {
		if cmp, ok := n.(Compound); ok {
			pID := mkID("t2", cmp)
			fmt.Fprintf(buf, "    %s [label=\"%s\", tooltip=\"weight: %d\"];\n", pID, p, n.Weight())
			for _, ch := range cmp.Children() {
				fmt.Fprintf(buf, "    %s -> %s;\n", pID, mkID("t2", ch))
			}
		}
	})
	fmt.Fprintf(buf, "  }\n")

	walk(t2, "", func(p string, n Node) {
		if cmp, ok := n.(Compound); ok {
			nID := mkID("t2", cmp)
			if cmp.Match() != nil {
				fmt.Fprintf(buf, "    %s -> %s[color=red,penwidth=1.0];\n", nID, mkID("t1", cmp.Match()))
			}
		}
	})

	fmt.Fprintf(buf, "}")

	ioutil.WriteFile("graph.dot", buf.Bytes(), os.ModePerm)
}

func TestBasicDiff(t *testing.T) {
	var a interface{}
	if err := json.Unmarshal([]byte(aJSON), &a); err != nil {
		panic(err)
	}

	var b interface{}
	if err := json.Unmarshal([]byte(bJSON), &b); err != nil {
		panic(err)
	}

	// Diff(a, b)
	ds, err := Diff(a, b)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.MarshalIndent(ds, "", "  ")
	fmt.Println(string(data))
}

type TestCase struct {
	description string
	src, dst    string // express test cases as json strings
	expect      []*Delta
}

func RunTestCases(t *testing.T, cases []TestCase) {
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

		diff, err := Diff(src, dst)
		if err != nil {
			t.Errorf("%d. '%s' unexpected error: %s", i, c.description, err)
			continue
		}
		if err := CompareDiffs(c.expect, diff); err != nil {
			t.Errorf("%d. '%s' result mismatch: %s", i, c.description, err)
		}
	}
}

func CompareDiffs(a, b []*Delta) error {
	if len(a) != len(b) {
		return fmt.Errorf("length mismatch: %d != %d", len(a), len(b))
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
		return fmt.Errorf("Type mismatch. %T != %T", a.Type, b.Type)
	}

	// TODO - compare SrcPaths & DstPaths

	if !reflect.DeepEqual(a.SrcVal, b.SrcVal) {
		return fmt.Errorf("SrcVal mismatch")
	}

	return nil
}

func TestQriUseCases(t *testing.T) {
	cases := []TestCase{
		{
			"detect scalar change",
			`[[0,1,2]]`,
			`[[0,1,3]]`,
			[]*Delta{
				{Type: DTChange, SrcPath: "0.2", DstPath: "0.2", SrcVal: 2, DstVal: 3},
			},
		},
		{
			"detect insert",
			`[[1]]`,
			`[[1],[2]]`,
			[]*Delta{
				{Type: DTInsert, SrcPath: "0.0", DstPath: "0.1", SrcVal: "", DstVal: []interface{}{2}},
			},
		},
		{
			"detect remove",
			`[[1],[2],[3]]`,
			`[[1],[3]]`,
			[]*Delta{
				{Type: DTRemove, SrcPath: "0.1", DstPath: "", SrcVal: []interface{}{2}, DstVal: nil},
			},
		},
		{
			"detect move",
			`[[1],[2],[3]]`,
			`[[1],[3],[2]]`,
			[]*Delta{
				{Type: DTMove, SrcPath: "0.1", DstPath: "0.2", SrcVal: []interface{}{2}, DstVal: []interface{}{2}},
			},
		},
	}

	RunTestCases(t, cases)
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
