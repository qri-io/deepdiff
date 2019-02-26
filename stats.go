package difff

// Stats holds statistical metadata about a diff
type Stats struct {
	Left  int `json:"leftNodes"`  // count of nodes in the left tree
	Right int `json:"rightNodes"` // count of nodes in the right tree

	LeftWeight  int `json:"leftWeight"`  // byte-ish count of left tree
	RightWeight int `json:"rightWeight"` // byte-ish count of right tree

	Inserts int `json:"inserts,omitempty"` // number of nodes inserted
	Updates int `json:"updates,omitempty"` // number of nodes updated
	Deletes int `json:"deletes,omitempty"` // number of nodes deleted
	Moves   int `json:"moves,omitempty"`   // number of nodes moved
}

// NodeChange returns a count of the shift between left & right trees
func (s Stats) NodeChange() int {
	return s.Right - s.Left
}

// PctWeightChange returns a value from -1.0 to max(float64) representing the size shift
// between left & right trees
func (s Stats) PctWeightChange() float64 {
	if s.RightWeight == 0 {
		// TODO (b5): better handle this unlikely scenario that may arise from misuse
		return 0
	}
	return float64(s.LeftWeight) / float64(s.RightWeight)
}
