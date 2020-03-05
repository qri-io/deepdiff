package deepdiff

import (
	"encoding/json"
	"reflect"
	"testing"
)

type PatchTestCase struct {
	description  string
	tree, expect interface{}
	patch        Deltas
}

func TestPatch(t *testing.T) {
	cases := []PatchTestCase{
		{
			"update bool",
			[]interface{}{true},
			[]interface{}{false},
			Deltas{{Type: DTUpdate, Path: IndexAddr(0), Value: false}},
		},
		{
			"update number",
			[]interface{}{float64(1)},
			[]interface{}{float64(2)},
			Deltas{{Type: DTUpdate, Path: IndexAddr(0), Value: float64(2)}},
		},
		{
			"update nested number",
			map[string]interface{}{"a": []interface{}{float64(1)}},
			map[string]interface{}{"a": []interface{}{float64(2)}},
			Deltas{{Type: DTContext, Path: StringAddr("a"), Deltas: Deltas{
				{Type: DTUpdate, Path: IndexAddr(0), Value: float64(2)}},
			}},
		},
		{
			"update string",
			[]interface{}{"before"},
			[]interface{}{"after"},
			Deltas{{Type: DTUpdate, Path: IndexAddr(0), Value: "after"}},
		},
		{
			"insert number to end of array",
			[]interface{}{},
			[]interface{}{float64(1)},
			Deltas{{Type: DTInsert, Path: IndexAddr(0), Value: float64(1)}},
		},
		{
			"insert number in slice",
			[]interface{}{float64(0), float64(2)},
			[]interface{}{float64(0), float64(1), float64(2)},
			Deltas{
				{Type: DTContext, Path: IndexAddr(0), Value: float64(0)},
				{Type: DTInsert, Path: IndexAddr(1), Value: float64(1)},
			},
		},
		{
			"insert false into object",
			map[string]interface{}{},
			map[string]interface{}{"a": false},
			Deltas{{Type: DTInsert, Path: StringAddr("a"), Value: false}},
		},
		{
			"delete from end of array",
			[]interface{}{"a", "b", "c"},
			[]interface{}{"a", "b"},
			Deltas{
				{Type: DTContext, Path: IndexAddr(0), Value: "a"},
				{Type: DTContext, Path: IndexAddr(1), Value: "b"},
				{Type: DTDelete, Path: IndexAddr(2), Value: "c"},
			},
		},
		{
			"delete from array",
			[]interface{}{"a", "b", "c"},
			[]interface{}{"a", "c"},
			Deltas{
				{Type: DTContext, Path: IndexAddr(0), Value: "a"},
				{Type: DTDelete, Path: IndexAddr(1), Value: "b"},
				{Type: DTContext, Path: IndexAddr(1), Value: "c"},
			},
		},
		{
			"delete from object",
			map[string]interface{}{"a": false},
			map[string]interface{}{},
			Deltas{
				{Type: DTDelete, Path: StringAddr("a")},
			},
		},
		{
			"delete from nested object",
			map[string]interface{}{
				"a": []interface{}{
					map[string]interface{}{
						"b": false,
					},
				},
			},
			map[string]interface{}{
				"a": []interface{}{
					map[string]interface{}{},
				},
			},
			Deltas{
				{Type: DTContext, Path: StringAddr("a"), Deltas: Deltas{
					{Type: DTContext, Path: IndexAddr(0), Deltas: Deltas{
						{Type: DTDelete, Path: StringAddr("b")},
					}},
				}},
			},
		},
		{
			"insert, update, then delete",
			map[string]interface{}{"a": true, "b": float64(2)},
			map[string]interface{}{"a": false, "c": float64(3)},
			Deltas{
				{Type: DTInsert, Path: StringAddr("c"), Value: float64(3)},
				{Type: DTUpdate, Path: StringAddr("a"), Value: false},
				{Type: DTDelete, Path: StringAddr("b"), Value: false},
			},
		},

		{
			"remove scalar from array in object",
			map[string]interface{}{"a": []interface{}{false, "yep"}, "b": true},
			map[string]interface{}{"a": []interface{}{"yep"}, "b": true},
			Deltas{
				{Type: DTContext, Path: StringAddr("a"), Deltas: Deltas{
					{Type: DTDelete, Path: IndexAddr(0), Value: false},
				}},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			if err := Patch(c.patch, &c.tree); err != nil {
				t.Fatalf("patch error: %s", err)
			}

			if !reflect.DeepEqual(c.tree, c.expect) {
				t.Errorf("result mismatch")
				if data, err := json.Marshal(c.tree); err == nil {
					t.Log("got   :", string(data))
				}
				if data, err := json.Marshal(c.expect); err == nil {
					t.Log("expect:", string(data))
				}
			}
		})
	}
}

type PatchErrorTestCase struct {
	description string
	tree        interface{}
	dlt         *Delta
	err         error
}

func RunPatchErrorTestCases(t *testing.T, cases []PatchErrorTestCase) {

}

func TestPatchErrors(t *testing.T) {
	errCases := []PatchErrorTestCase{}

	RunPatchErrorTestCases(t, errCases)
}
