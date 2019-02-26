package difff

import (
	"encoding/json"
	"reflect"
	"testing"
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
		Inserts:     3,
		Updates:     3,
		Deletes:     1,
		Moves:       0,
	}
	stats := &Stats{}
	Diff(a, b, OptionSetStats(stats))

	if expect.NodeChange() != stats.NodeChange() {
		t.Errorf("wrong node change. want: %d. got: %d", expect.NodeChange(), stats.NodeChange())
	}

	if expect.PctWeightChange() != stats.PctWeightChange() {
		t.Errorf("wrong percentage of node change. want: %f. got: %f", expect.PctWeightChange(), stats.PctWeightChange())
	}

	if !reflect.DeepEqual(expect, stats) {
		t.Errorf("response mismatch")
		t.Logf("want: %v", expect)
		t.Logf("got: %v", stats)
	}

}
