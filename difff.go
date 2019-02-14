// Package difff calculates the differences of document trees consisting of the
// standard go types created by unmarshaling from JSON, consisting of two
// complex types:
//   * map[string]interface{}
//   * []interface{}
// and five scalar types:
//   * string, int, float64, bool, nil
//
// difff is based off an algorithm designed for diffing XML documents outlined in:
//    Detecting Changes in XML Documents by Grégory Cobéna & Amélie Marian
//
// The paper describes an algorithm for generating an edit script that transitions
// between two states of tree-type data structures (XML). The general
// approach is as follows: For two given tree states, generate a diff script
// as a set of Deltas in 6 steps:
//
// 1. register in a map a unique signature (hash value) for every
//    subtree of the d1 (old) document
// 2. consider every subtree in d2 document, starting from the
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
package difff

import (
	"encoding/hex"
	"fmt"
	"hash"
	"hash/fnv"
	"sort"
	"strconv"
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
	dts := computeDeltas(t1, t2)

	return dts, nil
}

// DeltaType defines the types of changes xydiff can create
// to describe the difference between two documents
type DeltaType uint8

const (
	// DTUnknown defaults DeltaType to undefined behaviour
	DTUnknown DeltaType = iota
	// DTRemove means making the children of a node
	// become the children of a node's parent
	DTRemove
	// DTInsert is the compliment of deleting, adding
	// children of a parent node to a new node, and making
	// that node a child of the original parent
	DTInsert
	// DTMove is the succession of a deletion & insertion
	// of the same node
	DTMove
	// DTChange is an alteration of a scalar data type (string, bool, float, etc)
	DTChange
)

func (d DeltaType) String() string {
	switch d {
	case DTRemove:
		return "remove"
	case DTInsert:
		return "insert"
	case DTMove:
		return "move"
	case DTChange:
		return "change"
	default:
		return "unknown"
	}
}

// Delta represents a change between two documents
type Delta struct {
	Type DeltaType

	SrcPath string
	DstPath string

	SrcVal interface{}
	DstVal interface{}
}

func path(n Node) string {
	str := n.Name()
	for {
		n = n.Parent()
		if n == nil {
			break
		}
		str = fmt.Sprintf("%s.%s", n.Name(), str)
	}
	return str
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

type Node interface {
	Type() NodeType
	Hash() []byte
	Weight() int
	Parent() Node
	Name() string
	Value() interface{}
	Match() Node
	SetMatch(Node)
}

type Compound interface {
	Node
	Children() map[string]Node
	Child(key string) Node
}

type compound struct {
	t      NodeType
	name   string
	hash   []byte
	parent Node
	weight int
	value  interface{}
	match  Node

	children map[string]Node
}

func (c compound) Type() NodeType            { return c.t }
func (c compound) Name() string              { return c.name }
func (c compound) Hash() []byte              { return c.hash }
func (c compound) Weight() int               { return c.weight }
func (c compound) Parent() Node              { return c.parent }
func (c compound) Value() interface{}        { return c.value }
func (c compound) Match() Node               { return c.match }
func (c *compound) SetMatch(n Node)          { c.match = n }
func (c compound) Children() map[string]Node { return c.children }
func (c compound) Child(name string) Node    { return c.children[name] }

type scalar struct {
	t      NodeType
	name   string
	hash   []byte
	parent Node
	value  interface{}
	weight int
	match  Node
}

func (s scalar) Type() NodeType     { return s.t }
func (s scalar) Name() string       { return s.name }
func (s scalar) Hash() []byte       { return s.hash }
func (s scalar) Weight() int        { return s.weight }
func (s scalar) Parent() Node       { return s.parent }
func (s scalar) Value() interface{} { return s.value }
func (s scalar) Match() Node        { return s.match }
func (s *scalar) SetMatch(n Node)   { s.match = n }

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
		arr := &compound{
			t:        NTArray,
			name:     name,
			parent:   parent,
			children: map[string]Node{},
			value:    v,
		}

		for i, v := range x {
			name := strconv.Itoa(i)
			node := tree(v, name, arr, nodes)
			hasher.Write(node.Hash())
			arr.children[name] = node
		}
		arr.hash = hasher.Sum(nil)

		arr.weight = 1
		for _, ch := range arr.children {
			arr.weight += ch.Weight()
		}
		n = arr
	case map[string]interface{}:
		hasher := NewHash()
		obj := &compound{
			t:        NTObject,
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
					// fmt.Printf("matching: %s and %s\n", path(n2), path(cp))
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
	// fmt.Println("dist", dist, "maxDist:", maxDist, "n2Weight:", n2.Weight(), "n2path:", path(n2), "treeWeight:", t2Weight)
}

func optimize() {

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
		}
	}
}

func propagateMatchToChildren(n Node) {
	// if a node is matched & a compound type,
	if n1, ok := n.(Compound); ok && n.Match() != nil {
		if n2, ok := n.Match().(Compound); ok {
			if n1.Type() == NTObject && n2.Type() == NTObject {
				// match any key names
				for name, n1ch := range n1.Children() {
					if n2ch := n2.Child(name); n2ch != nil {
						n2ch.SetMatch(n1ch)
					}
				}
			}
			if n1.Type() == NTArray && n2.Type() == NTArray && len(n1.Children()) == len(n2.Children()) {
				// if arrays are the same length, match all children
				// b/c these are arrays, no names should be missing, safe to skip check
				for name, n1ch := range n1.Children() {
					n2.Child(name).SetMatch(n1ch)
				}
			}
		}
	}
}

func computeDeltas(t1, t2 Node) []*Delta {
	ds := calcICDs(t1, t2)
	calcMoves(ds)
	organize(ds)
	return ds
}

func walk(tree Node, path string, fn func(path string, n Node)) {
	path = fmt.Sprintf("%s.%s", path, tree.Name())
	fn(path, tree)
	if cmp, ok := tree.(Compound); ok {
		for _, n := range cmp.Children() {
			walk(n, path, fn)
		}
	}
}

// calculate inserts, changes, & deletes
func calcICDs(t1, t2 Node) (dts []*Delta) {
	walk(t1, "", func(path string, n Node) {
		if t1.Match() == nil {
			delta := &Delta{
				Type:    DTRemove,
				SrcPath: path,
				SrcVal:  n.Value(),
			}
			dts = append(dts, delta)
		}
	})

	walk(t2, "", func(path string, n Node) {
		if t1.Match() == nil {
			delta := &Delta{
				Type:    DTInsert,
				DstPath: path,
				DstVal:  n.Value(),
			}
			dts = append(dts, delta)
		}
	})
	return dts
}

func calcMoves(ds []*Delta) {

}

func organize([]*Delta) {
}

// lcss calculates the longest common subsequence
func lcss() {

}
