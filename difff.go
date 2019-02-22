package difff

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/fnv"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

type DiffConfig struct {
	MoveDeltas bool
}

type DiffOption func(cfg *DiffConfig)

// Diff computes a slice of deltas that define an edit script for turning the
// value at d1 into d2
func Diff(d1, d2 interface{}, opts ...DiffOption) []*Delta {
	cfg := &DiffConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	difff := &diff{cfg: cfg, d1: d1, d2: d2}
	return difff.diff()
}

type diff struct {
	cfg     *DiffConfig
	d1, d2  interface{}
	t1, t2  Node
	t1Nodes map[string][]Node
}

func (d *diff) diff() []*Delta {
	d.t1, d.t2, d.t1Nodes = d.prepTrees()
	d.queueMatch(d.t1Nodes, d.t2)
	d.optimize(d.t1, d.t2)
	d.optimize(d.t1, d.t2)
	return d.computeDeltas(d.t1, d.t2)
}

func walk(tree Node, path string, fn func(path string, n Node) bool) {
	if tree.Name() != "" {
		path += fmt.Sprintf("/%s", tree.Name())
	}
	kontinue := fn(path, tree)
	if cmp, ok := tree.(Compound); kontinue && ok {
		for _, n := range cmp.Children() {
			walk(n, path, fn)
		}
	}
}

func walkPostfix(tree Node, path string, fn func(path string, n Node)) {
	if tree.Name() != "" {
		path += fmt.Sprintf("/%s", tree.Name())
	}
	if cmp, ok := tree.(Compound); ok {
		for _, n := range cmp.Children() {
			walkPostfix(n, path, fn)
		}
	}
	fn(path, tree)
}

func path(n Node) string {
	var path []string
	for {
		if n == nil || n.Name() == "" {
			break
		}
		path = append([]string{n.Name()}, path...)
		n = n.Parent()
	}
	return "/" + strings.Join(path, "/")
}

// NewHash returns a new hash interface, wrapped in a function for easy
// hash algorithm switching, package consumers can override NewHash
// with their own desired hash.Hash implementation if the value space is
// particularly large. default is 32-bit FNV 1 for fast, cheap hashing
var NewHash = func() hash.Hash {
	return fnv.New64()
}

func hashStr(sum []byte) string {
	return hex.EncodeToString(sum)
}

func (d *diff) queueMatch(t1Nodes map[string][]Node, t2 Node) {
	queue := make(chan Node)
	done := make(chan struct{})
	considering := 1
	t2Weight := t2.Weight()

	go func() {
		var candidates []Node
		for n2 := range queue {
			key := hashStr(n2.Hash())

			candidates = t1Nodes[key]

			switch len(candidates) {
			case 0:
				// no candidates. check if node has children. If so, add them.
				if n2c, ok := n2.(Compound); ok {
					for _, ch := range n2c.Children() {
						considering++
						go func(n Node) {
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
func matchNodes(n1, n2 Node) {
	n1.SetMatch(n2)
	n2.SetMatch(n1)
	n1p := n1.Parent()
	n2p := n2.Parent()
	for n1p != nil && n2p != nil {
		// TODO - root name is coming back as "", need to think about why this is
		// and weather it's ok to match roots
		if n1p.Name() == n2p.Name() && n1p.Name() != "" && n2p.Name() != "" {
			// fmt.Printf("also matching %s %s %s %d\n", path(n1p), path(n2p), n1p.Name(), n1p.Type())
			n1p.SetMatch(n2p)
			n2p.SetMatch(n1p)
			n1p = n1p.Parent()
			n2p = n2p.Parent()
		}
		break
	}
}

// bestCandidate is the one who's parent
func bestCandidate(t1Candidates []Node, n2 Node, t2Weight int) {
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

func (d *diff) optimize(t1, t2 Node) {
	// var wg sync.WaitGroup
	walkPostfix(t1, "", func(p string, n Node) {
		// wg.Add(1)
		// go func() {
		propagateMatchToParent(n)
		propagateMatchToChildren(n)
		// wg.Done()
		// }()
	})
	// wg.Wait()
	walkPostfix(t2, "", func(p string, n Node) {
		// wg.Add(1)
		// go func() {
		propagateMatchToParent(n)
		propagateMatchToChildren(n)
		// wg.Done()
		// }()
	})
	// wg.Wait()
}

func propagateMatchToParent(n Node) {
	// if n is a compound type that isn't matched
	if cmp, ok := n.(Compound); ok && n.Match() == nil {
		var match Node
		// iterate each child
		for _, ch := range cmp.Children() {
			// if this child has a match
			if m := ch.Match(); m != nil && m.Parent() != nil {
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
			n.SetMatch(match)
			match.SetMatch(n)
		}
	}
}

func propagateMatchToChildren(n Node) {
	// if a node is matched & a compound type,
	if n1, ok := n.(Compound); ok && n.Match() != nil {
		if n2, ok := n.Match().(Compound); ok {
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
				// b/c these are arrays, no names should be missing, safe to skip check
				for _, n1ch := range n1.Children() {
					n2ch := n2.Child(n1ch.Name())
					n2ch.SetMatch(n1ch)
					n1ch.SetMatch(n2ch)
				}
			}
		}
	}
}

func (d *diff) computeDeltas(t1, t2 Node) []*Delta {
	ds := d.calcDeltas(t1, t2)
	return ds
}

// calculate inserts, changes, deletes, & moves
func (d *diff) calcDeltas(t1, t2 Node) (dts []*Delta) {
	// fmt.Println("calc moves?", d.cfg.MoveDeltas)
	walk(t1, "", func(p string, n Node) bool {
		if n.Match() == nil {
			delta := &Delta{
				Type:    DTDelete,
				SrcPath: p,
				SrcVal:  n.Value(),
			}
			dts = append(dts, delta)

			// update t1 array values to reflect deletion so later comparisons will be
			// accurate. only place where this really applies is parent of delete is
			// an array (object paths will remain accurate)
			if parent := n.Parent(); parent != nil {
				if cmp, ok := parent.(Compound); ok && cmp.Type() == ntArray {
					idx64, err := strconv.ParseInt(n.Name(), 0, 0)
					if err != nil {
						panic(err)
					}
					idx := int(idx64)
					for i, n := range cmp.Children() {
						if i > idx {
							n.SetName(strconv.Itoa(i - 1))
						}
					}
				}
			}

			// at this point we have the most general insert possible. By
			// returning false here we stop traversing to any existing children
			// avoiding redundant inserts already described by the parent
			return false
		}
		return true
	})

	var parentMoves []*Delta
	walk(t2, "", func(p string, n Node) bool {
		match := n.Match()
		if match == nil {
			delta := &Delta{
				Type:    DTInsert,
				DstPath: p,
				DstVal:  n.Value(),
			}
			dts = append(dts, delta)

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
					for i, n := range match.(Compound).Children() {
						if i > idx {
							n.SetName(strconv.Itoa(i + 1))
						}
					}
				}
			}

			// at this point we have the most general insert possible. By
			// returning false here we stop traversing to any existing children
			// avoiding redundant inserts already described by the parent
			return false
		}

		if d.cfg.MoveDeltas {
			// If we have a match & parents are different, this corresponds to a move
			if path(match.Parent()) != path(n.Parent()) {
				delta := &Delta{
					Type:    DTMove,
					DstPath: p,
					SrcPath: path(match),
					SrcVal:  match.Value(),
					DstVal:  n.Value(),
				}
				dts = append(dts, delta)
				parentMoves = append(parentMoves, delta)

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
						for i, n := range match.(Compound).Children() {
							if i > idx {
								n.SetName(strconv.Itoa(i + 1))
							}
						}
					}
				}

				// break matching to prevent connection later on
				match.Parent().SetMatch(nil)
				n.Parent().SetMatch(nil)
				// match.SetMatch(nil)
				// n.SetMatch(nil)

				return false
			}
		}

		if _, ok := n.(Compound); !ok {
			// check if value is scalar, creating a change delta if so
			// TODO (b5): this needs to be a check to see if it's a leaf node
			// (eg, empty object is a leaf node)
			if delta := compareScalar(match, n, p); delta != nil {
				dts = append(dts, delta)
			}
		}
		return true
	})

	if d.cfg.MoveDeltas {
		var cleanups []string
		walk(t2, "", func(p string, n Node) bool {
			if n.Type() == ntArray && n.Match() != nil {
				// matches to same array-type parent require checking for shuffles within the parent
				// *expensive*
				deltas := calcReorderDeltas(n.Match().(Compound).Children(), n.(Compound).Children())
				for _, d := range deltas {
					cleanups = append(cleanups, d.SrcPath, d.DstPath)
				}
				if deltas != nil {
					dts = append(dts, deltas...)
					return false
				}
			}
			return true
		})

		var cleaned []*Delta
	CLEANUP:
		for _, d := range dts {
			for _, pth := range cleanups {
				if d.Type == DTUpdate && (strings.HasPrefix(d.SrcPath, pth) || strings.HasPrefix(d.DstPath, pth)) {
					continue CLEANUP
				}
			}
			cleaned = append(cleaned, d)
		}
		return cleaned
	}

	return dts
}

// calcReorderDeltas creates deltas that describes moves within the same parent
// it starts by calculates the largest (order preserving) common subsequence between
// two matched parent Compound nodes
// https://en.wikipedia.org/wiki/Longest_common_subsequence_problem
//
func calcReorderDeltas(a, b []Node) (deltas []*Delta) {
	var wg sync.WaitGroup
	max := len(a)
	if len(b) > max {
		max = len(b)
	}
	aRem := len(a) - 1
	bRem := len(b) - 1
	pageSize := 50

	for i := 0; i <= max/pageSize; i++ {
		var aPage, bPage []Node
		start := (i * pageSize)
		// fmt.Println(start, start+pageSize, a, b)
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
		go func(a, b []Node) {
			if ds := movedBNodes(a, b); ds != nil {
				deltas = append(deltas, ds...)
			}
			wg.Done()
		}(aPage, bPage)
	}
	wg.Wait()

	return
}

func movedBNodes(allA, allB []Node) []*Delta {
	var a, b []Node
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
		// fmt.Printf("%d\n", i)
		c[i] = make([]int, n)
		for j := 1; j < n; j++ {
			if a[i-1].Match() != nil && b[j-1].Match() != nil {
				// reflect.DeepEqual(a[i-1].Value(), b[j-1].Value())
				// a[i-1].Name() == b[j-1].Name()
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

	var ass, bss []Node
	backtrackB(&ass, c, a, b, m-1, n-1)
	backtrackA(&bss, c, a, b, m-1, n-1)
	amv := intersect(a, ass)
	bmv := intersect(b, bss)

	var deltas []*Delta
	for i := 0; i < len(amv); i++ {
		am := amv[i]
		bm := bmv[i]

		// don't add moves that have the same source & destination paths
		// can be created by matches that move between parents
		if path(am) != path(bm) {
			mv := &Delta{
				Type:    DTMove,
				SrcPath: path(am),
				DstPath: path(bm),
				DstVal:  bm.Value(),
			}
			deltas = append(deltas, mv)
		}
	}

	return deltas
}

// intersect produces a set intersection, assuming subset is a subset of set and both nodes are ordered
func intersect(set, subset []Node) (nodes []Node) {
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

// function backtrack(C[0..m,0..n], X[1..m], Y[1..n], i, j)
//   if i = 0 or j = 0
//       return ""
//   if  X[i] = Y[j]
//       return backtrack(C, X, Y, i-1, j-1) + X[i]
//   if C[i,j-1] > C[i-1,j]
//       return backtrack(C, X, Y, i, j-1)
//   return backtrack(C, X, Y, i-1, j)
func backtrackA(ss *[]Node, c [][]int, a, b []Node, i, j int) {
	if i == 0 || j == 0 {
		return
	}

	if bytes.Equal(a[i-1].Hash(), b[j-1].Hash()) {
		// TODO (b5): I think this is where we can backtrack based on which node
		// has the greater weight by taking, need to check
		// if b[j].Weight() > a[i].Weight() {
		// fmt.Printf("append %p, %s\n", b[j-1], path(b[j-1]))
		*ss = append([]Node{a[i-1]}, *ss...)
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

func backtrackB(ss *[]Node, c [][]int, a, b []Node, i, j int) {
	if i == 0 || j == 0 {
		return
	}

	if bytes.Equal(a[i-1].Hash(), b[j-1].Hash()) {
		// TODO (b5): I think this is where we can backtrack based on which node
		// has the greater weight by taking, need to check
		// if b[j].Weight() > a[i].Weight() {
		// fmt.Printf("append %p, %s\n", b[j-1], path(b[j-1]))
		*ss = append([]Node{b[j-1]}, *ss...)
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

func compareScalar(n1, n2 Node, n2Path string) *Delta {
	if n1.Type() != n2.Type() {
		return &Delta{
			Type:    DTUpdate,
			DstPath: n2Path,
			SrcVal:  n1.Value(),
			DstVal:  n2.Value(),
		}
	}
	if !reflect.DeepEqual(n1.Value(), n2.Value()) {
		return &Delta{
			Type:    DTUpdate,
			SrcPath: path(n1),
			DstPath: n2Path,
			SrcVal:  n1.Value(),
			DstVal:  n2.Value(),
		}
	}
	return nil
}
