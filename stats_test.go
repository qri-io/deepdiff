package deepdiff

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCalcStats(t *testing.T) {
	aJSON := []byte(`{"a": 100,"foo": [1,2,3],"bar": false,"baz": {"a": {"b": 4,"c": false,"d": "apples-and-oranges"},"e": null,"g": "apples-and-oranges"}}`)
	bJSON := []byte(`{"a": 99,"foo": [1,2,3],"bar": false,"baz": {"a": {"b": 5,"c": false,"d": "apples-and-oranges"},"e": "thirty-thousand-something-dogecoin","f": {"a" : false, "b": true}}}`)

	var a, b map[string]interface{}
	if err := json.Unmarshal(aJSON, &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(bJSON, &b); err != nil {
		t.Fatal(err)
	}

	expect := &Stats{
		Left:        14,
		Right:       16,
		LeftWeight:  186,
		RightWeight: 268,
		Inserts:     6,
		Updates:     0,
		Deletes:     4,
		Moves:       0,
	}

	got, err := NewDeepDiff().Stat(context.Background(), a, b)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}
}
