package deepdiff

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDeltaJSON(t *testing.T) {
	patch := `[[" ", "apples", {"foo": false}, [["-", 2, false]] ]]`
	var dts Deltas
	if err := json.Unmarshal([]byte(patch), &dts); err != nil {
		t.Fatal(err)
	}

	expect := Deltas{
		{Type: DTContext, Path: StringAddr("apples"), Value: map[string]interface{}{"foo": false}, Deltas: Deltas{
			{Type: DTDelete, Path: IndexAddr(2), Value: false},
		}},
	}

	if diff := cmp.Diff(expect, dts); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}
