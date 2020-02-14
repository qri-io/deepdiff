package deepdiff

import (
	"bytes"
	"context"
	"encoding/hex"
	"hash"
	"hash/fnv"
	"reflect"
	"sort"
	"strconv"
	"sync"
)

// Config are any possible configuration parameters for calculating diffs
type Config struct {
	// If true Diff will calculate "moves" that describe changing the parent of
	// a subtree
	MoveDeltas bool
	// Setting Changes to true will have diff represent in-place value shifts
	// as changes instead of add-delete pairs
	Changes bool
}

// DiffOption is a function that adjust a config, zero or more DiffOptions
// can be passed to the Diff function
type DiffOption func(cfg *Config)

// DeepDiff is a configuration for performing diffs
type DeepDiff struct {
	changes    bool
	moveDeltas bool
}

// New creates a deepdiff struct
func New(opts ...DiffOption) *DeepDiff {
	cfg := &Config{}
	for _, opt := range opts {
		opt(cfg)
	}

	return &DeepDiff{
		changes:    cfg.Changes,
		moveDeltas: cfg.MoveDeltas,
	}
}

// Diff computes a slice of deltas that define an edit script for turning a
// into b.
// currently Diff will never return an error, error returns are reserved for
// future use. specifically: bailing before delta calculation based on a
// configurable threshold
func (dd *DeepDiff) Diff(ctx context.Context, a, b interface{}) (Deltas, error) {
	deepdiff := &diff{moves: dd.moveDeltas, changes: dd.changes, d1: a, d2: b}
	return deepdiff.diff(ctx), nil
}

// StatDiff calculates a diff script and diff stats
func (dd *DeepDiff) StatDiff(ctx context.Context, a, b interface{}) (Deltas, *Stats, error) {
	deepdiff := &diff{moves: dd.moveDeltas, changes: dd.changes, d1: a, d2: b, stats: &Stats{}}
	return deepdiff.diff(ctx), deepdiff.stats, nil
}

// Stat calculates the DiffStata between two documents
func (dd *DeepDiff) Stat(ctx context.Context, a, b interface{}) (*Stats, error) {
	deepdiff := &diff{moves: dd.moveDeltas, changes: dd.changes, d1: a, d2: b, stats: &Stats{}}
	deepdiff.diff(ctx)
	return deepdiff.stats, nil
}

// diff is a state machine for calculating an edit script that transitions
// between two state trees
type diff struct {
	moves   bool // calculate moves flag
	changes bool // calculate changes flag
	stats   *Stats
	d1, d2  interface{}
	t1, t2  node
	t1Nodes map[string][]node
}

// diff calculates a structl diff for two given tree states
// generating an edit script as a list of Delta changes:
//
// 1. prepTrees - register in a map a unique signature (hash value) for every
//    subtree of the d1 (old) document
// 2. queueMatch - consider every subtree in d2 document, starting from the
//    largest. check if it is identitical to some the subtrees in
//    d1, if so match both subtrees.
// 3. attempt to match the parents of two matched subtrees
//    by checking labels (in our case, types of parent object or array)
//    controlling for bad matches based on length of path to the
//    ancestor and the weight of the matching subtrees. eg: a large
//    subtree may force the matching of its ancestors up to the root
//    a small subtree may not even force matching of its parent
// 4. Consider the largest subtrees of d2 in order. If one candidate
//    has it's parent already matched to the parent of the considered
//    node, it is certianly the best candidate.
// 5. At this point we might have matched all of d2. A node may not
//    match b/c its been inserted, or we missed matching it. We can now
//    do peephole optimization pass to retry some of the rejected nodes
//    once no more matchings can be obtained, unmatched nodes in d2
//    correspond to inserted nodes.
// 6. consider each matching node and decide if the node is at its right
//    place, or whether it has been moved.
func (d *diff) diff(ctx context.Context) Deltas {
	d.t1, d.t2, d.t1Nodes = d.prepTrees(ctx)
	d.queueMatch(d.t1Nodes, d.t2)
	d.optimize(d.t1, d.t2)
	// TODO (b5): a second optimize pass seems to help greatly on larger diffs, which
	// to me seems we should propagating matches more aggressively in the optimize pass,
	// removing the need for this second call (which is effectively doing the same
	// thing as recursive/aggressive match propagation)
	d.optimize(d.t1, d.t2)
	d.optimize(d.t1, d.t2)
	return d.calcDeltas(d.t1, d.t2)
}

// NewHash returns a new hash interface, wrapped in a function for easy
// hash algorithm switching, package consumers can override NewHash
// with their own desired hash.Hash implementation if the value space is
// particularly large. default is 64-bit FNV 1 for fast, cheap,
// (non-cryptographic) hashing
var NewHash = func() hash.Hash {
	return fnv.New64()
}

// hashString converts a hash sum to a string using hex encoding
// localized here for easy ecnsoding swapping
func hashStr(sum []byte) string {
	return hex.EncodeToString(sum)
}

func (d *diff) queueMatch(t1Nodes map[string][]node, t2 node) {
	queue := make(chan node)
	done := make(chan struct{})
	considering := 1
	t2Weight := t2.Weight()

	go func() {
		var candidates []node
		for n2 := range queue {
			key := hashStr(n2.Hash())

			candidates = t1Nodes[key]

			switch len(candidates) {
			case 0:
				// no candidates. check if node has children. If so, add them.
				if n2c, ok := n2.(compound); ok {
					for _, ch := range n2c.Children() {
						considering++
						go func(n node) {
							queue <- n
						}(ch)
					}
				}
			case 1:
				// connect an exact match. yay!
				n1 := candidates[0]
				matchNodes(n1, n2)
			default:
				// choose a best candidate. let the sketchiness begin.
				bestCandidate(candidates, n2, t2Weight)
			}

			considering--
			if considering == 0 {
				done <- struct{}{}
				break
			}
		}
	}()

	// start queue with t2 (root of tree)
	queue <- t2
	<-done
	return
}

// matchNodes connects two nodes & tries to propagate that match upward to
// ancestors so long as labels match
func matchNodes(n1, n2 node) {
	n1.SetMatch(n2)
	n2.SetMatch(n1)
	n1p := n1.Parent()
	n2p := n2.Parent()
	for n1p != nil && n2p != nil {
		if n1p.Name() == n2p.Name() && n1p.Name() != "" && n2p.Name() != "" {
			n1p.SetMatch(n2p)
			n2p.SetMatch(n1p)
			n1p = n1p.Parent()
			n2p = n2p.Parent()
		}
		break
	}
}

func bestCandidate(t1Candidates []node, n2 node, t2Weight int) {
	maxDist := 1 + float32(n2.Weight())/float32(t2Weight)
	dist := 1 + float32(n2.Parent().Weight()-n2.Weight())/float32(t2Weight)
	n2 = n2.Parent()

	for dist < maxDist {
		for i, can := range t1Candidates {
			if cp := can.Parent(); cp != nil {
				if n2.Name() == cp.Name() {
					matchNodes(cp, n2)
					return
				}
			}
			t1Candidates[i] = can.Parent()
		}
		if n2.Parent() == nil {
			break
		}
		dist = 1 + float32(n2.Parent().Weight()-n2.Weight())/float32(t2Weight)
		n2 = n2.Parent()
	}
}

func (d *diff) optimize(t1, t2 node) {
	walkPostfix(t1, nil, func(p []string, n node) {
		propagateMatchToParent(n)
	})
	walkPostfix(t2, nil, func(p []string, n node) {
		propagateMatchToParent(n)
	})
	walk(t1, nil, func(p []string, n node) bool {
		propagateMatchToChildren(n)
		return true
	})
	walk(t2, nil, func(p []string, n node) bool {
		propagateMatchToChildren(n)
		return true
	})
}

func propagateMatchToParent(n node) {
	// if n is a compound type that isn't matched
	if cmp, ok := n.(compound); ok && n.Match() == nil {
		var match node
		// iterate each child
		for _, ch := range cmp.Children() {
			// if this child has a match, and the matches parent doesn't have a match,
			// match the parents
			if m := ch.Match(); m != nil && m.Parent() != nil && m.Parent().Match() == nil {
				p := m.Parent()
				if match == nil {
					match = p
				} else if p.Weight() > m.Weight() {
					// if a match already exists, keep the heavier match
					match = p
				}
			}
		}
		if match != nil {
			matchNodes(match, n)
		}
	}
}

func propagateMatchToChildren(n node) {
	// if a node is matched & a compound type,
	if n1, ok := n.(compound); ok && n.Match() != nil {
		if n2, ok := n.Match().(compound); ok {
			if n1.Type() == ntObject && n2.Type() == ntObject {
				// match any key names
				for _, n1ch := range n1.Children() {
					if n2ch := n2.Child(n1ch.Name()); n2ch != nil {
						n2ch.SetMatch(n1ch)
					}
				}
			}
			if n1.Type() == ntArray && n2.Type() == ntArray && len(n1.Children()) == len(n2.Children()) {
				// if arrays are the same length, match all children
				// b/c these are arrays, no names should be missing, safe to skip a name check
				for _, n1ch := range n1.Children() {
					n2ch := n2.Child(n1ch.Name())
					n2ch.SetMatch(n1ch)
					n1ch.SetMatch(n2ch)
				}
			}
		}
	}
}

// calculate inserts, deletes, and/or changes & moves by folding tree A into
// tree B, adding unmatched nodes from A to B as deletes
func (d *diff) calcDeltas(t1, t2 node) (dts Deltas) {

	// fold t1 into t2, adding deletes to t2
	walkSorted(t1, nil, func(p []string, n node) bool {
		if n.Match() == nil {
			n.SetChangeType(DTDelete)

			if cmp, ok := n.(compound); ok {
				cmp.DropChildNodes()
			}

			// delta := &Delta{
			// 	Type:  DTDelete,
			// 	Path:  p[len(p)-1],
			// 	Value: n.Value(),
			// }
			addNode(t2, n, p)
			// dts = append(dts, delta)

			// update t1 array values to reflect deletion so later comparisons will be
			// accurate. only place where this really applies is parent of delete is
			// an array (object paths will remain accurate)
			if parent := n.Parent(); parent != nil {
				if arr, ok := parent.(*array); ok {
					idx64, err := strconv.ParseInt(n.Name(), 0, 0)
					if err != nil {
						panic(err)
					}
					idx := int(idx64)
					for i, n := range arr.Children() {
						name := strconv.Itoa(i - 1)
						arr.childNames[name] = i - 1
						if i > idx {
							n.SetName(name)
						}
					}
				}
			}

			// at this point we have the most general insert we know of
			if cmp, ok := n.(compound); ok {
				// drop any childs node references so subsequent iterations of the
				// tree to build up a delta don't iterate any deeper
				cmp.DropChildNodes()
			}

			// by returning false here we stop traversing to any existing children
			// avoiding redundant inserts already described by the parent
			return false
		}
		return true
	})

	// var parentMoves Deltas
	walkSorted(t2, nil, func(p []string, n node) bool {
		// at this point deletes from t1 have been moved here, need to skip 'em
		// because n.Match will be a circular reference
		if n.ChangeType() == DTDelete {
			return false
		}

		match := n.Match()
		if match == nil {
			n.SetChangeType(DTInsert)
			// delta := &Delta{
			// 	Type:  DTInsert,
			// 	Path:  p[len(p)-1],
			// 	Value: n.Value(),
			// }
			// dts = append(dts, delta)

			// update t1 array values to reflect insertion so later comparisons will be
			// accurate. only place where this really applies is parent of insert is
			// an array (object paths will remain accurate)
			if parent := n.Parent(); parent != nil && parent.Type() == ntArray {
				if match, ok := parent.Match().(*array); ok && match != nil {
					idx64, err := strconv.ParseInt(n.Name(), 0, 0)
					if err != nil {
						panic(err)
					}
					idx := int(idx64)
					for i, n := range match.Children() {
						name := strconv.Itoa(i + 1)
						match.childNames[name] = i + 1
						if i > idx {
							n.SetName(name)
						}
					}
				}
			}

			// at this point we have the most general insert we know of
			if cmp, ok := n.(compound); ok {
				// drop any childs node references so subsequent iterations of the
				// tree to build up a delta don't iterate any deeper
				cmp.DropChildNodes()
			}

			// By returning false here we stop traversing to any existing children
			// avoiding redundant inserts already described by the parent
			return false
		}

		if d.moves {
			// If we have a match & parents are different, this corresponds to a move
			if pathString(path(match.Parent())) != pathString(path(n.Parent())) {
				// delta := &Delta{
				// 	Type:        DTMove,
				// 	Path:        p[len(p)-1],
				// 	Value:       n.Value(),
				// 	SourcePath:  pathString(path(match)),
				// 	SourceValue: match.Value(),
				// }
				// dts = append(dts, delta)
				// parentMoves = append(parentMoves, delta)
				n.SetChangeType(DTMove)

				// update t1 array values to reflect insertion so later comparisons will be
				// accurate. only place where this really applies is parent of insert is
				// an array (object paths will remain accurate)
				if parent := n.Parent(); parent != nil && parent.Type() == ntArray {
					if match := parent.Match(); match != nil {
						idx64, err := strconv.ParseInt(n.Name(), 0, 0)
						if err != nil {
							panic(err)
						}
						idx := int(idx64)
						for i, n := range match.(compound).Children() {
							if i > idx {
								n.SetName(strconv.Itoa(i + 1))
							}
						}
					}
				}

				// break matching to prevent connection later on
				match.Parent().SetMatch(nil)
				n.Parent().SetMatch(nil)

				return false
			}
		}

		if _, ok := n.(compound); !ok {
			// check if value is scalar, creating a change delta if so
			// TODO (b5): this needs to be a check to see if it's a leaf node
			// (eg, empty object is a leaf node)
			if delta := compareScalar(match, n, p[len(p)-1]); delta != nil {
				n.SetChangeType(DTUpdate)
				// if d.changes {
				// 	// addDelta(root, delta, p)
				// 	dts = append(dts, delta)
				// } else {
				// 	// addDelta(root, &Delta{Type: DTDelete, Path: p[len(p)-1], Value: delta.SourceValue}, p)
				// 	// addDelta(root, &Delta{Type: DTInsert, Path: p[len(p)-1], Value: delta.Value}, p)
				// 	// dts = append(dts,
				// 	// 	&Delta{Type: DTDelete, Path: delta.Path, Value: delta.SourceValue},
				// 	// 	&Delta{Type: DTInsert, Path: delta.Path, Value: delta.Value},
				// 	// )
				// }
			}
		}
		return true
	})

	// if d.moves {
	// 	var cleanups []string
	// 	walkSorted(t2, nil, func(p []string, n node) bool {
	// 		if n.Type() == ntArray && n.Match() != nil {
	// 			// matches to same array-type parent require checking for shuffles within the parent
	// 			// *expensive*
	// 			deltas := calcReorderDeltas(n.Match().(compound).Children(), n.(compound).Children())
	// 			for _, d := range deltas {
	// 				cleanups = append(cleanups, d.SourcePath, d.Path)
	// 			}
	// 			if deltas != nil {
	// 				dts = append(dts, deltas...)
	// 				return false
	// 			}
	// 		}
	// 		return true
	// 	})

	// 	var cleaned Deltas
	// CLEANUP:
	// 	for _, d := range dts {
	// 		for _, pth := range cleanups {
	// 			if d.Type == DTUpdate && (strings.HasPrefix(d.SourcePath, pth) || strings.HasPrefix(d.Path, pth)) {
	// 				continue CLEANUP
	// 			}
	// 		}
	// 		cleaned = append(cleaned, d)
	// 	}
	// 	return cleaned
	// }

	script, _ := d.childDeltas(t2.(compound))
	sortDeltasAndMaybeCalcStats(script, d.stats)

	return script
}

func (d *diff) childDeltas(cmp compound) (changes Deltas, hasChanges bool) {
	ch := cmp.Children()

	for _, n := range ch {
		dlt := toDelta(n)
		if dlt.Type == DTContext {
			if childCmp, ok := n.(compound); ok {
				if children, hasChanges := d.childDeltas(childCmp); hasChanges {
					dlt.Value = nil
					dlt.Deltas = children
				}
			}
		} else {
			hasChanges = true
		}

		// If we aren't outputting changes, convert to a delete/insert combo
		if dlt.Type == DTUpdate && !d.changes {
			changes = append(changes, &Delta{Type: DTDelete, Path: dlt.Path, Value: dlt.SourceValue})
			dlt.Type = DTInsert
			dlt.SourceValue = nil
		}

		changes = append(changes, dlt)
	}

	return changes, hasChanges
}

func sortDeltasAndMaybeCalcStats(deltas Deltas, st *Stats) {
	sort.Sort(deltas)

	for _, d := range deltas {
		if len(d.Deltas) > 0 {
			sortDeltasAndMaybeCalcStats(d.Deltas, st)
		}

		if st != nil {
			switch d.Type {
			case DTInsert:
				st.Inserts++
			case DTUpdate:
				st.Updates++
			case DTDelete:
				st.Deletes++
			case DTMove:
				st.Moves++
			}
		}
	}
}

// calcReorderDeltas creates deltas that describes moves within the same parent
// it starts by calculates the largest (order preserving) common subsequence between
// two matched parent compound nodes. Background on LCSS:
// https://en.wikipedia.org/wiki/Longest_common_subsequence_problem
//
// reorder calculation is shingled into sets of maximum 50 values & processed parallel
// to keep things fast at the expense of missing some common sequences from longer lists
func calcReorderDeltas(a, b []node) (deltas Deltas) {
	var wg sync.WaitGroup
	max := len(a)
	if len(b) > max {
		max = len(b)
	}
	aRem := len(a) - 1
	bRem := len(b) - 1
	pageSize := 50

	for i := 0; i <= max/pageSize; i++ {
		var aPage, bPage []node
		start := (i * pageSize)
		if (start + pageSize) > aRem {
			aPage = a[start:]
		} else {
			aPage = a[start : start+pageSize]
		}

		if (start + pageSize) > bRem {
			bPage = b[start:]
		} else {
			bPage = b[start : start+pageSize]
		}

		wg.Add(1)
		go func(a, b []node) {
			if ds := movedBNodes(a, b); ds != nil {
				deltas = append(deltas, ds...)
			}
			wg.Done()
		}(aPage, bPage)
	}
	wg.Wait()

	return
}

func movedBNodes(allA, allB []node) Deltas {
	var a, b []node
	for _, n := range allA {
		if n.Match() != nil {
			a = append(a, n)
		}
	}

	for _, n := range allB {
		if n.Match() != nil {
			b = append(b, n)
		}
	}

	m := len(a) + 1
	n := len(b) + 1
	c := make([][]int, m)
	c[0] = make([]int, n)

	for i := 1; i < m; i++ {
		c[i] = make([]int, n)
		for j := 1; j < n; j++ {
			if a[i-1].Match() != nil && b[j-1].Match() != nil {
				if bytes.Equal(a[i-1].Hash(), b[j-1].Hash()) {
					c[i][j] = c[i-1][j-1] + 1
				} else {
					c[i][j] = c[i][j-1]
					if c[i-1][j] > c[i][j] {
						c[i][j] = c[i-1][j]
					}
				}
			}
		}
	}

	// TODO (b5): a & b *should* be the same length, which would mean a bottom-right
	// common-value that's equal to the length of a should mean list equality
	// which means we need to bail early b/c no moves exist
	if c[m-1][n-1] == len(a) || c[m-1][n-1] == len(b) {
		return nil
	}

	var ass, bss []node
	backtrackB(&ass, c, a, b, m-1, n-1)
	backtrackA(&bss, c, a, b, m-1, n-1)
	amv := intersect(a, ass)
	bmv := intersect(b, bss)

	var deltas Deltas
	for i := 0; i < len(amv); i++ {
		am := amv[i]
		bm := bmv[i]

		// don't add moves that have the same source & destination paths
		// can be created by matches that move between parents
		if pathString(path(am)) != pathString(path(bm)) {
			mv := &Delta{
				Type:       DTMove,
				Path:       pathString(path(bm)),
				Value:      bm.Value(),
				SourcePath: pathString(path(am)),
			}
			deltas = append(deltas, mv)
		}
	}

	return deltas
}

// intersect produces a set intersection, assuming subset is a subset of set and both nodes are ordered
func intersect(set, subset []node) (nodes []node) {
	if len(set) == len(subset) {
		return nil
	}

	c := 0

SET:
	for _, n := range set {
		if c == len(subset) {
			nodes = append(nodes, set[c:]...)
			break
		}

		for _, ssn := range subset[c:] {
			if bytes.Equal(n.Hash(), ssn.Hash()) {
				c++
				continue SET
			}
		}

		nodes = append(nodes, n)
	}

	return
}

// backtrack walks the "a" side of a common sequence matrix backward, constructing the
// secuence of nodes from the "a" (lefthand) node list
func backtrackA(ss *[]node, c [][]int, a, b []node, i, j int) {
	if i == 0 || j == 0 {
		return
	}

	if bytes.Equal(a[i-1].Hash(), b[j-1].Hash()) {
		// TODO (b5): I think this is where we can backtrack based on which node
		// has the greater weight by taking different paths in the commonalitiy index matrix
		// need to check...
		// if b[j].Weight() > a[i].Weight() {
		// fmt.Printf("append %p, %s\n", b[j-1], path(b[j-1]))
		*ss = append([]node{a[i-1]}, *ss...)
		// } else {
		// ss = append(ss, a[i])
		// }
		backtrackA(ss, c, a, b, i-1, j-1)
		return
	}
	if c[i][j-1] > c[i-1][j] {
		backtrackA(ss, c, a, b, i, j-1)
		return
	}

	backtrackA(ss, c, a, b, i-1, j)
	return
}

// backtrack walks the "b" side of a common sequence matrix backward, constructing the
// secuence of nodes from the "b" (righthand) node list
func backtrackB(ss *[]node, c [][]int, a, b []node, i, j int) {
	if i == 0 || j == 0 {
		return
	}

	if bytes.Equal(a[i-1].Hash(), b[j-1].Hash()) {
		// TODO (b5): I think this is where we can backtrack based on which node
		// has the greater weight by taking different paths in the commonalitiy index matrix
		// need to check...
		// if b[j].Weight() > a[i].Weight() {
		// fmt.Printf("append %p, %s\n", b[j-1], path(b[j-1]))
		*ss = append([]node{b[j-1]}, *ss...)
		// } else {
		// ss = append(ss, a[i])
		// }
		backtrackB(ss, c, a, b, i-1, j-1)
		return
	}
	if c[i][j-1] > c[i-1][j] {
		backtrackB(ss, c, a, b, i, j-1)
		return
	}

	backtrackB(ss, c, a, b, i-1, j)
	return
}

// compareScalar compares two scalar values, possibly creating an Update delta
func compareScalar(n1, n2 node, n2Path string) *Delta {
	if n1.Type() != n2.Type() {
		return &Delta{
			Type:        DTUpdate,
			Path:        n2Path,
			Value:       n2.Value(),
			SourceValue: n1.Value(),
		}
	}
	if !reflect.DeepEqual(n1.Value(), n2.Value()) {
		return &Delta{
			Type:        DTUpdate,
			Path:        n2Path,
			Value:       n2.Value(),
			SourceValue: n1.Value(),
		}
	}
	return nil
}

func toDelta(n node) *Delta {
	d := &Delta{Type: n.ChangeType(), Path: n.Name()}
	if string(d.Type) == "" {
		d.Type = DTContext
	}

	switch d.Type {
	case DTUpdate:
		d.Value = n.Value()
		d.SourceValue = n.Match().Value()
	case DTInsert, DTDelete, DTContext:
		d.Value = n.Value()
	}

	return d
}
