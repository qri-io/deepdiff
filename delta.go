package deepdiff

import (
	"encoding/json"
	"strconv"
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

// Addr is a single location within a data structure. Multiple path elements can
// be stitched together into a single
type Addr interface {
	json.Marshaler
	Value() interface{}
	String() string
	Eq(a Addr) bool
}

// StringAddr is an arbitrary key representation within a data structure.
// Most-often used to represent map keys
type StringAddr string

// Value returns StringAddr as a string, a common go type
func (p StringAddr) Value() interface{} {
	return string(p)
}

// String returns this address as a string
func (p StringAddr) String() string {
	return string(p)
}

// Eq tests for equality with another address
func (p StringAddr) Eq(b Addr) bool {
	sa, ok := b.(StringAddr)
	if !ok {
		return false
	}

	return p == sa
}

// MarshalJSON implements the json.Marshaller interface
func (p StringAddr) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(p))
}

// IndexAddr is the address of a location within list-type structures
type IndexAddr int

// Value returns IndexAddr as an int, a common go type
func (p IndexAddr) Value() interface{} {
	return int(p)
}

// String returns this address as a string
func (p IndexAddr) String() string {
	return strconv.Itoa(int(p))
}

// Eq tests for equality with another address
func (p IndexAddr) Eq(b Addr) bool {
	sa, ok := b.(IndexAddr)
	if !ok {
		return false
	}

	return p == sa
}

// MarshalJSON implements the json.Marshaller interface
func (p IndexAddr) MarshalJSON() ([]byte, error) {
	return json.Marshal(int(p))
}

// RootAddr is a nihlic address, or a reference to the outside address space
type RootAddr struct{}

// Value of RootAddr is nil
func (RootAddr) Value() interface{} {
	return nil
}

// String root address is "/" by convention
func (RootAddr) String() string {
	return "/"
}

// Eq checks for Root Address equality
func (RootAddr) Eq(b Addr) bool {
	_, ok := b.(RootAddr)
	if !ok {
		return false
	}

	return true
}

// MarshalJSON writes "null". RootAddress is represented as null in JSON
func (RootAddr) MarshalJSON() ([]byte, error) {
	return json.Marshal(nil)
}

type sortableAddrs []Addr

func (a sortableAddrs) Len() int { return len(a) }
func (a sortableAddrs) Less(i,j int) bool {
	if ii, ok := a[i].Value().(int); ok {
		if jj, ok := a[j].Value().(int); ok {
			return ii < jj
		}
	}

	return a[i].String() < a[j].String()
}
func (a sortableAddrs) Swap(i,j int) {
	a[i], a[j] = a[j], a[i]
}

// Delta represents a change between a source & destination document
// a delta is a single "edit" that describes changes to the destination document
type Delta struct {
	// the type of change
	Type Operation `json:"type"`
	// Path is a string representation of the patch to where the delta operation
	// begins in the destination documents
	// path should conform to the IETF JSON-pointer specification, outlined
	// in RFC 6901: https://tools.ietf.org/html/rfc6901
	Path Addr `json:"path"`
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
// come *before* inserts in diff script presentation
var opOrder = map[Operation]uint{
	DTDelete:  0,
	DTContext: 1,
	DTInsert:  2,
	DTUpdate:  3,
}

// Less returns true if the value at index i is a smaller sort quantity than
// the value at index j
func (ds Deltas) Less(i, j int) bool {
	iAddr := ds[i].Path
	jAddr := ds[j].Path
	if iAddr.Eq(jAddr) {
		return opOrder[ds[i].Type] < opOrder[ds[j].Type]
	}

	if a, ok := ds[i].Path.Value().(int); ok {
		if b, ok := ds[j].Path.Value().(int); ok {
			return a < b 
		}
	}

	return iAddr.String() < jAddr.String()
}

// Swap reverses the values at indicies i & J
func (ds Deltas) Swap(i, j int) { ds[i], ds[j] = ds[j], ds[i] }
