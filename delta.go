package deepdiff

import (
	"encoding/json"
)

// Operation defines the operation of a Delta item
type Operation string

const (
	// DTContext indicates unchanged contextual details present in both A and B
	DTContext = Operation(" ")
	// DTDelete means making the children of a node
	// become the children of a node's parent
	DTDelete = Operation("-")
	// DTInsert is the compliment of deleting, adding
	// children of a parent node to a new node, and making
	// that node a child of the original parent
	DTInsert = Operation("+")
	// DTUpdate is an alteration of a scalar data type (string, bool, float, etc)
	DTUpdate = Operation("~")
)

// Delta represents a change between a source & destination document
// a delta is a single "edit" that describes changes to the destination document
type Delta struct {
	// the type of change
	Type Operation `json:"type"`
	// Path is a string representation of the patch to where the delta operation
	// begins in the destination documents
	// path should conform to the IETF JSON-pointer specification, outlined
	// in RFC 6901: https://tools.ietf.org/html/rfc6901
	Path string `json:"path"`
	// The value to change in the destination document
	Value interface{} `json:"value"`

	// To make delta's revesible, original values are included
	// the original path this change from
	SourcePath string `json:"SourcePath,omitempty"`
	// the original  value this was changed from, will not always be present
	SourceValue interface{} `json:"originalValue,omitempty"`

	// Child Changes
	Deltas `json:"deltas,omitempty"`
}

// MarshalJSON implements a custom JOSN Marshaller
func (d *Delta) MarshalJSON() ([]byte, error) {
	v := []interface{}{d.Type, d.Path}
	if len(d.Deltas) > 0 {
		v = append(v, nil, d.Deltas)
	} else {
		v = append(v, d.Value)
	}
	return json.Marshal(v)
}

// Deltas is a sortable slice of changes
type Deltas []*Delta

// Len returns the length of the slice
func (ds Deltas) Len() int { return len(ds) }

// opOrder determines a total order for operations. It's crucial that deletes
// comve *before* inserts in diff script presentation
var opOrder = map[Operation]uint{
	DTDelete:  0,
	DTContext: 1,
	DTInsert:  2,
	DTUpdate:  3,
}

// Less returns trus if the value at index i is a smaller sort quantity than
// the value at index j
func (ds Deltas) Less(i, j int) bool {
	return ds[i].Path < ds[j].Path || (ds[i].Path == ds[j].Path && opOrder[ds[i].Type] < opOrder[ds[j].Type])
}

// Swap reverses the values at indicies i & J
func (ds Deltas) Swap(i, j int) { ds[i], ds[j] = ds[j], ds[i] }
