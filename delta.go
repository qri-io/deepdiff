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
	// DTMove is the succession of a deletion & insertion
	// of the same node
	DTMove = Operation(">")
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
	Deltas []*Delta `json:"deltas,omitempty"`
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
