package difff

// Operation defines the operation of a Delta item
type Operation string

const (
	// DTDelete means making the children of a node
	// become the children of a node's parent
	DTDelete = Operation("delete")
	// DTInsert is the compliment of deleting, adding
	// children of a parent node to a new node, and making
	// that node a child of the original parent
	DTInsert = Operation("insert")
	// DTMove is the succession of a deletion & insertion
	// of the same node
	DTMove = Operation("move")
	// DTUpdate is an alteration of a scalar data type (string, bool, float, etc)
	DTUpdate = Operation("update")
)

// Delta represents a change between two documents
type Delta struct {
	Type Operation

	SrcPath string
	DstPath string

	SrcVal interface{}
	DstVal interface{}
}
