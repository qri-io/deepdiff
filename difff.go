package difff

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/fnv"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var (
	// ErrCompletelyDistinct is a holdover error while we work out the details
	ErrCompletelyDistinct = fmt.Errorf("these things are totally different")
)

// Diff computes a slice of deltas that define an edit script for turning the
// value at d1 into d2
func Diff(d1, d2 interface{}) ([]*Delta, error) {
	t1, t2, t1Nodes := prepTrees(d1, d2)
	queueMatch(t1Nodes, t2)
	optimize(t1, t2)
	dts := computeDeltas(t1, t2)

	return dts, nil
}

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

func rootNode(n Node) Node {
	for {
		if n.Parent() == nil {
			return n
		}
		n = n.Parent()
	}
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

// NodeType defines all of the atoms in our universe
type NodeType uint8

const (
	// NTUnknown defines a type outside our universe, should never be encountered
	NTUnknown NodeType = iota
	// NTObject is a dictionary of key / value pairs
	NTObject
	NTArray
	NTString
	NTFloat
	NTInt
	NTBool
	NTNull
)

func (nt NodeType) String() string {
	switch nt {
	case NTObject:
		return "Object"
	case NTArray:
		return "Array"
	case NTString:
		return "String"
	case NTFloat:
		return "Float"
	case NTInt:
		return "Int"
	case NTBool:
		return "Bool"
	case NTNull:
		return "Null"
	default:
		return "Unknown"
	}
}

type Node interface {
	Type() NodeType
	Hash() []byte
	Weight() int
	Parent() Node
	Name() string
	SetName(string)
	Value() interface{}
	Match() Node
	SetMatch(Node)
}

type Compound interface {
	Node
	Children() []Node
	Child(key string) Node
}

type object struct {
	name   string
	hash   []byte
	parent Node
	weight int
	value  interface{}
	match  Node

	children map[string]Node
}

func (o object) Type() NodeType       { return NTObject }
func (o object) Name() string         { return o.name }
func (o *object) SetName(name string) { o.name = name }
func (o object) Hash() []byte         { return o.hash }
func (o object) Weight() int          { return o.weight }
func (o object) Parent() Node         { return o.parent }
func (o object) Value() interface{}   { return o.value }
func (o object) Match() Node          { return o.match }
func (o *object) SetMatch(n Node)     { o.match = n }
func (o object) Children() []Node {
	nodes := make([]Node, len(o.children))
	i := 0
	for _, ch := range o.children {
		nodes[i] = ch
		i++
	}
	return nodes
}
func (o object) Child(name string) Node { return o.children[name] }

type array struct {
	name   string
	hash   []byte
	parent Node
	weight int
	value  interface{}
	match  Node

	children []Node
}

func (c array) Type() NodeType       { return NTArray }
func (c array) Name() string         { return c.name }
func (c *array) SetName(name string) { c.name = name }
func (c array) Hash() []byte         { return c.hash }
func (c array) Weight() int          { return c.weight }
func (c array) Parent() Node         { return c.parent }
func (c array) Value() interface{}   { return c.value }
func (c array) Match() Node          { return c.match }
func (c *array) SetMatch(n Node)     { c.match = n }
func (c array) Children() []Node     { return c.children }
func (c array) Child(name string) Node {
	for _, ch := range c.children {
		if ch.Name() == name {
			return ch
		}
	}
	return nil
}

type scalar struct {
	t      NodeType
	name   string
	hash   []byte
	parent Node
	value  interface{}
	weight int
	match  Node
}

func (s scalar) Type() NodeType       { return s.t }
func (s scalar) Name() string         { return s.name }
func (s *scalar) SetName(name string) { s.name = name }
func (s scalar) Hash() []byte         { return s.hash }
func (s scalar) Weight() int          { return s.weight }
func (s scalar) Parent() Node         { return s.parent }
func (s scalar) Value() interface{}   { return s.value }
func (s scalar) Match() Node          { return s.match }
func (s *scalar) SetMatch(n Node)     { s.match = n }

func prepTrees(d1, d2 interface{}) (t1, t2 Node, t1Nodes map[string][]Node) {
	var (
		wg        sync.WaitGroup
		t1nodesCh = make(chan Node)
		t2nodesCh = make(chan Node)
	)

	t1Nodes = map[string][]Node{}
	wg.Add(2)

	go func(nodes <-chan Node) {
		for n := range nodes {
			key := hashStr(n.Hash())
			t1Nodes[key] = append(t1Nodes[key], n)
		}
		wg.Done()
	}(t1nodesCh)
	go func() {
		t1 = tree(d1, "", nil, t1nodesCh)
		close(t1nodesCh)
	}()

	go func(nodes <-chan Node) {
		for range nodes {
			// do nothing
		}
		wg.Done()
	}(t2nodesCh)
	go func() {
		t2 = tree(d2, "", nil, t2nodesCh)
		close(t2nodesCh)
	}()

	wg.Wait()
	return
}

func tree(v interface{}, name string, parent Node, nodes chan Node) (n Node) {
	switch x := v.(type) {
	case nil:
		n = &scalar{
			t:      NTNull,
			name:   name,
			hash:   NewHash().Sum([]byte("null")),
			parent: parent,
			value:  v,
			weight: 1,
		}
	case float64:
		fstr := strconv.FormatFloat(x, 'f', -1, 64)
		n = &scalar{
			t:      NTFloat,
			name:   name,
			hash:   NewHash().Sum([]byte(fstr)),
			parent: parent,
			value:  v,
			weight: len(fstr),
		}
	case string:
		n = &scalar{
			t:      NTString,
			name:   name,
			hash:   NewHash().Sum([]byte(x)),
			parent: parent,
			value:  v,
			weight: len(x),
		}
	case bool:
		bstr := "false"
		if x {
			bstr = "true"
		}
		n = &scalar{
			t:      NTBool,
			name:   name,
			hash:   NewHash().Sum([]byte(bstr)),
			parent: parent,
			value:  v,
			weight: len(bstr),
		}
	case []interface{}:
		hasher := NewHash()
		arr := &array{
			name:     name,
			parent:   parent,
			children: make([]Node, len(x)),
			value:    v,
		}

		for i, v := range x {
			name := strconv.Itoa(i)
			node := tree(v, name, arr, nodes)
			hasher.Write(node.Hash())
			arr.children[i] = node
		}
		arr.hash = hasher.Sum(nil)

		arr.weight = 1
		for _, ch := range arr.children {
			arr.weight += ch.Weight()
		}
		n = arr
	case map[string]interface{}:
		hasher := NewHash()
		obj := &object{
			name:     name,
			parent:   parent,
			children: map[string]Node{},
			value:    v,
		}

		// gotta sort keys for consistent hashing :(
		names := make([]string, 0, len(x))
		for name := range x {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			node := tree(x[name], name, obj, nodes)
			hasher.Write(node.Hash())
			obj.children[name] = node
		}
		obj.hash = hasher.Sum(nil)

		obj.weight = 1
		for _, ch := range obj.children {
			obj.weight += ch.Weight()
		}
		n = obj
	default:
		panic(fmt.Sprintf("unexpected type: %T", v))
	}

	nodes <- n
	return
}

// sortAdd inserts n into nodes, keeping the slice sorted by node weight,
// heaviest to the left
func sortAdd(n Node, nodes []Node) []Node {
	i := sort.Search(len(nodes), func(i int) bool { return nodes[i].Weight() <= n.Weight() })
	nodes = append(nodes, nil)
	copy(nodes[i+1:], nodes[i:])
	nodes[i] = n
	return nodes
}

func queueMatch(t1Nodes map[string][]Node, t2 Node) {
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

func optimize(t1, t2 Node) {
	walk(t1, "", func(p string, n Node) bool {
		propagateMatchToParent(n)
		propagateMatchToChildren(n)
		return true
	})
	walk(t2, "", func(p string, n Node) bool {
		propagateMatchToParent(n)
		propagateMatchToChildren(n)
		return true
	})
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
			if n1.Type() == NTObject && n2.Type() == NTObject {
				// match any key names
				for _, n1ch := range n1.Children() {
					if n2ch := n2.Child(n1ch.Name()); n2ch != nil {
						n2ch.SetMatch(n1ch)
					}
				}
			}
			if n1.Type() == NTArray && n2.Type() == NTArray && len(n1.Children()) == len(n2.Children()) {
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

func computeDeltas(t1, t2 Node) []*Delta {
	ds := calcDeltas(t1, t2)
	return ds
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

// calculate inserts, changes, deletes, & moves
func calcDeltas(t1, t2 Node) (dts []*Delta) {
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
				if cmp, ok := parent.(Compound); ok && cmp.Type() == NTArray {
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
			if parent := n.Parent(); parent != nil && parent.Type() == NTArray {
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
			if parent := n.Parent(); parent != nil && parent.Type() == NTArray {
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

	var cleanups []string
	walk(t2, "", func(p string, n Node) bool {
		if n.Type() == NTArray && n.Match() != nil {
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
			if d.Type == DTChange && (strings.HasPrefix(d.SrcPath, pth) || strings.HasPrefix(d.DstPath, pth)) {
				continue CLEANUP
			}
		}
		cleaned = append(cleaned, d)
	}

	return cleaned
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
		// if _, ok := n.(Compound); ok {
		if n.Match() != nil {
			a = append(a, n)
		}
		// }
	}

	for _, n := range allB {
		// if _, ok := n.(Compound); ok {
		if n.Match() != nil {
			b = append(b, n)
		}
		// }
	}

	m := len(a) + 1
	n := len(b) + 1
	c := make([][]int, m)
	c[0] = make([]int, n)
	// fmt.Println("-- moved --", len(a), len(b))

	for i := 1; i < m; i++ {
		// fmt.Printf("%d\n", i)
		c[i] = make([]int, n)
		for j := 1; j < n; j++ {
			// fmt.Printf("%p %p | %p %p\n", a[i-1], b[j-1], a[i-1].Match(), b[j-1].Match())
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

	// for i := 0; i < m; i++ {
	// 	fmt.Printf("  | ")
	// 	for j := 0; j < n; j++ {
	// 		fmt.Printf("%d ", c[i][j])
	// 	}
	// 	fmt.Println("|")
	// }

	var ass, bss []Node
	backtrackB(&ass, c, a, b, m-1, n-1)
	backtrackA(&bss, c, a, b, m-1, n-1)
	amv := intersect(a, ass)
	bmv := intersect(b, bss)

	// fmt.Println("matches:")
	// for _, n := range a {
	// 	fmt.Printf("(addr: %p name: %s value: %v, match: %p) ", n, n.Name(), n.Value(), n.Match())
	// }
	// fmt.Println("")

	// for _, n := range b {
	// 	fmt.Printf("(addr: %p name: %s value: %v, match: %p) ", n, n.Name(), n.Value(), n.Match())
	// }
	// fmt.Println("")

	// fmt.Printf("%v %v | %v %v\n", a, ass, bss, b)
	// fmt.Printf("%v | %v\n", amv, bmv)

	// fmt.Println("move deltas:")
	var deltas []*Delta
	for i := 0; i < len(amv); i++ {
		am := amv[i]
		bm := bmv[i]

		// don't add moves that have the same source & destination paths
		// can be created by matches that move between parents
		if path(am) != path(bm) {
			// fmt.Printf("A:(addr: %p name: %s value: %v, match: %p)\n", am, am.Name(), am.Value(), am.Match())
			// fmt.Printf("B:(addr: %p name: %s value: %v, match: %p)\n", bm, bm.Name(), bm.Value(), bm.Match())

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
			Type:    DTChange,
			DstPath: n2Path,
			SrcVal:  n1.Value(),
			DstVal:  n2.Value(),
		}
	}
	if !reflect.DeepEqual(n1.Value(), n2.Value()) {
		return &Delta{
			Type:    DTChange,
			SrcPath: path(n1),
			DstPath: n2Path,
			SrcVal:  n1.Value(),
			DstVal:  n2.Value(),
		}
	}
	return nil
}
