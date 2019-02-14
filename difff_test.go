package difff

import (
	"encoding/json"
	"fmt"
	"reflect"
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

func TestBasicDiff(t *testing.T) {
	var a interface{}
	if err := json.Unmarshal([]byte(aJSON), &a); err != nil {
		panic(err)
	}

	var b interface{}
	if err := json.Unmarshal([]byte(bJSON), &b); err != nil {
		panic(err)
	}

	Diff(a, b)
}

type TestCase struct {
	description string
	src, dst    string // express test cases as json strings
	expect      []Delta
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

func CompareDiffs(a, b []Delta) error {
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

func CompareDeltas(a, b Delta) error {
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
			[]Delta{
				{Type: DTChange, SrcPath: []string{"0", "2"}, DstPath: []string{"0", "2"}, SrcVal: 2, DstVal: 3},
			},
		},
		{
			"detect insert",
			`[[1]]`,
			`[[1],[2]]`,
			[]Delta{
				{Type: DTInsert, SrcPath: []string{"0", "0"}, DstPath: []string{"0", "1"}, SrcVal: nil, DstVal: []interface{}{2}},
			},
		},
		{
			"detect remove",
			`[[1],[2],[3]]`,
			`[[1],[3]]`,
			[]Delta{
				{Type: DTRemove, SrcPath: []string{"0", "1"}, DstPath: nil, SrcVal: []interface{}{2}, DstVal: nil},
			},
		},
		{
			"detect move",
			`[[1],[2],[3]]`,
			`[[1],[3],[2]]`,
			[]Delta{
				{Type: DTMove, SrcPath: []string{"0", "1"}, DstPath: []string{"0", "2"}, SrcVal: []interface{}{2}, DstVal: []interface{}{2}},
			},
		},
	}

	RunTestCases(t, cases)
}

func BenchmarkDiff(b *testing.B) {
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
