package difff

// DeltaType defines the types of changes xydiff can create
// to describe the difference between two documents
type DeltaType string

const (
	// DTDelete means making the children of a node
	// become the children of a node's parent
	DTDelete = DeltaType("delete")
	// DTInsert is the compliment of deleting, adding
	// children of a parent node to a new node, and making
	// that node a child of the original parent
	DTInsert = DeltaType("insert")
	// DTMove is the succession of a deletion & insertion
	// of the same node
	DTMove = DeltaType("move")
	// DTChange is an alteration of a scalar data type (string, bool, float, etc)
	DTChange = DeltaType("change")
)

// Delta represents a change between two documents
type Delta struct {
	Type DeltaType

	SrcPath string
	DstPath string

	SrcVal interface{}
	DstVal interface{}
}
