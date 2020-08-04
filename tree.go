package deepdiff

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
)

// nodeType defines all of the atoms in our universe, or the types of data we
// will encounter while generating a diff
type nodeType uint8

const (
	ntUnknown nodeType = iota
	ntObject
	ntArray
	ntString
	ntFloat
	ntInt
	ntBool
	ntNull
)

// node represents a value in a tree for diff computation
type node interface {
	Type() nodeType
	// a byte hash of this node's content & any child nodes
	Hash() []byte
	//
	Weight() int
	// this node's parent, if one existts
	Parent() node
	// the name this parent has given this node. for arrays this'll be the string
	// value of this node's index, for objects this will be the key
	Addr() Addr
	// assign this node's name, only needs to be used when re-ordering nodes
	// post-deletion & insertion
	SetAddr(Addr)
	// the actual data this node is created from
	Value() interface{}
	// this node's counterpart in another node tree
	Match() node
	// assign this node's counterpart
	SetMatch(node)

	// node modification type accessor
	ChangeType() (o Operation)
	// assign a modification type to this node
	SetChangeType(Operation)
}

// compound represents a data type that can contain children
// basically objects & arrays
type compound interface {
	node
	// list children, for objects this will come in random order
	Children() []node
	// get a child by address
	Child(address Addr) node
	// add a child node. when calling this it's important to never add a child
	// that already exists
	AddChild(n node)
	// how many descendants this node has
	DescendantsCount() int
	// release all references to child nodes. note that the _value_ still contains
	// the data this node represents
	DropChildNodes()
}

// nodes implements the sort interface for a slice of nodes
type nodes []node

func (ns nodes) Len() int           { return len(ns) }
func (ns nodes) Less(i, j int) bool { return ns[i].Addr().String() < ns[j].Addr().String() }
func (ns nodes) Swap(i, j int)      { ns[i], ns[j] = ns[j], ns[i] }

type object struct {
	addr   Addr
	hash   []byte
	parent node
	weight int
	value  interface{}
	match  node
	change Operation

	descendants int
	children    map[Addr]node
}

func (o object) Type() nodeType              { return ntObject }
func (o object) Addr() Addr                { return o.addr }
func (o *object) SetAddr(addr Addr)        { o.addr = addr }
func (o object) Hash() []byte                { return o.hash }
func (o object) Weight() int                 { return o.weight }
func (o object) Parent() node                { return o.parent }
func (o object) Value() interface{}          { return o.value }
func (o object) Match() node                 { return o.match }
func (o *object) SetMatch(n node)            { o.match = n }
func (o *object) SetChangeType(dt Operation) { o.change = dt }
func (o *object) ChangeType() Operation      { return o.change }
func (o object) Children() []node {
	nodes := make([]node, len(o.children))
	i := 0
	for _, ch := range o.children {
		nodes[i] = ch
		i++
	}
	return nodes
}
func (o object) Child(a Addr) node { return o.children[a] }
func (o *object) AddChild(n node) {
	if cmp, ok := n.(compound); ok {
		o.descendants += cmp.DescendantsCount()
	}
	o.descendants++
	o.children[n.Addr()] = n
}
func (o *object) DropChildNodes()      { o.children = nil }
func (o object) DescendantsCount() int { return o.descendants }

type array struct {
	addr   Addr
	hash   []byte
	parent node
	weight int
	value  interface{}
	match  node
	change Operation

	descendants int
	childNames  map[Addr]int
	children    []node
}

func (c array) Type() nodeType              { return ntArray }
func (c array) Addr() Addr                { return c.addr }
func (c *array) SetAddr(addr Addr)        { c.addr = addr }
func (c array) Hash() []byte                { return c.hash }
func (c array) Weight() int                 { return c.weight }
func (c array) Parent() node                { return c.parent }
func (c array) Value() interface{}          { return c.value }
func (c array) Match() node                 { return c.match }
func (c *array) SetMatch(n node)            { c.match = n }
func (c *array) ChangeType() Operation      { return c.change }
func (c *array) SetChangeType(dt Operation) { c.change = dt }
func (c array) Children() []node            { return c.children }
func (c array) Child(addr Addr) node {
	if c.childNames[addr] < len(c.children) {
		return c.children[c.childNames[addr]]
	}
	return nil
}
func (c *array) AddChild(n node) {
	if cmp, ok := n.(compound); ok {
		c.descendants += cmp.DescendantsCount()
	}
	c.descendants++
	c.children = append(c.children, n)
}
func (c *array) DropChildNodes()      { c.children = nil }
func (c array) DescendantsCount() int { return c.descendants }

type scalar struct {
	t      nodeType
	addr   Addr
	hash   []byte
	parent node
	value  interface{}
	weight int
	match  node
	change Operation
}

func (s scalar) Type() nodeType              { return s.t }
func (s scalar) Addr() Addr                { return s.addr }
func (s *scalar) SetAddr(addr Addr)        { s.addr = addr }
func (s scalar) Hash() []byte                { return s.hash }
func (s scalar) Weight() int                 { return s.weight }
func (s scalar) Parent() node                { return s.parent }
func (s scalar) Value() interface{}          { return s.value }
func (s scalar) Match() node                 { return s.match }
func (s *scalar) SetMatch(n node)            { s.match = n }
func (s *scalar) ChangeType() Operation      { return s.change }
func (s *scalar) SetChangeType(dt Operation) { s.change = dt }

func (d *diff) prepTrees(ctx context.Context) (t1, t2 node, t1nodes map[string][]node) {
	var (
		wg                sync.WaitGroup
		t1nodesCh         = make(chan node)
		t2nodesCh         = make(chan node)
		t1Count, t1Weight int
		t2Count, t2Weight int
	)

	t1nodes = map[string][]node{}
	wg.Add(2)

	go func(nodes <-chan node) {
		for n := range nodes {
			key := hashStr(n.Hash())
			t1nodes[key] = append(t1nodes[key], n)
			t1Count++
			t1Weight += n.Weight()
		}
		wg.Done()
	}(t1nodesCh)
	go func() {
		t1 = tree(d.d1, RootAddr{}, nil, t1nodesCh)
		close(t1nodesCh)
	}()

	go func(nodes <-chan node) {
		for n := range nodes {
			// do nothing
			t2Count++
			t2Weight += n.Weight()
		}
		wg.Done()
	}(t2nodesCh)
	go func() {
		t2 = tree(d.d2, RootAddr{}, nil, t2nodesCh)
		close(t2nodesCh)
	}()

	wg.Wait()

	if d.stats != nil {
		d.stats.Left = t1Count
		d.stats.LeftWeight = t1Weight
		d.stats.Right = t2Count
		d.stats.RightWeight = t2Weight
	}
	return
}

func tree(v interface{}, addr Addr, parent node, nodes chan node) (n node) {
	v = preprocessType(v)
	switch x := v.(type) {
	case nil:
		n = &scalar{
			t:      ntNull,
			addr:   addr,
			hash:   NewHash().Sum([]byte("null")),
			parent: parent,
			value:  v,
			weight: 1,
		}
	case int64:
		istr := strconv.FormatInt(x, 10)
		n = &scalar{
			t:      ntInt,
			addr:   addr,
			hash:   NewHash().Sum([]byte(istr)),
			parent: parent,
			value:  v,
			weight: len(istr),
		}
	case float64:
		fstr := strconv.FormatFloat(x, 'f', -1, 64)
		n = &scalar{
			t:      ntFloat,
			addr:   addr,
			hash:   NewHash().Sum([]byte(fstr)),
			parent: parent,
			value:  v,
			weight: len(fstr),
		}
	case string:
		n = &scalar{
			t:      ntString,
			addr:   addr,
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
			t:      ntBool,
			addr:   addr,
			hash:   NewHash().Sum([]byte(bstr)),
			parent: parent,
			value:  v,
			weight: len(bstr),
		}
	case []interface{}:
		hasher := NewHash()
		arr := &array{
			addr:       addr,
			parent:     parent,
			childNames: map[Addr]int{},
			children:   make([]node, len(x)),
			value:      v,
		}

		for i, v := range x {
			node := tree(v, IndexAddr(i), arr, nodes)
			hasher.Write(node.Hash())
			arr.childNames[IndexAddr(i)] = i
			arr.children[i] = node

			if cmp, ok := node.(compound); ok {
				arr.descendants += cmp.DescendantsCount()
			}
			arr.descendants++
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
			addr:     addr,
			parent:   parent,
			children: map[Addr]node{},
			value:    v,
		}

		// gotta sort keys for consistent hashing :(
		addrs := make(sortableAddrs, 0, len(x))
		for name := range x {
			addrs = append(addrs, StringAddr(name))
		}
		sort.Sort(addrs)

		for _, addr := range addrs {
			node := tree(x[addr.String()], addr, obj, nodes)
			hasher.Write(node.Hash())
			obj.children[addr] = node

			if cmp, ok := node.(compound); ok {
				obj.descendants += cmp.DescendantsCount()
			}
			obj.descendants++
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

func preprocessType(v interface{}) interface{} {
	switch x := v.(type) {
	case map[interface{}]interface{}:
		conv := map[string]interface{}{}
		for key, val := range x {
			conv[fmt.Sprintf("%v", key)] = val
		}
		return conv
	case []string:
		conv := make([]interface{}, len(x))
		for i, s := range x {
			conv[i] = s
		}
		return conv
	case uint8:
		return int(x)
	case uint16:
		return int(x)
	case uint32:
		return int(x)
	case float32:
		return float64(x)
	default:
		return v
	}
}

// path computes the string path from
func path(n node) []Addr {
	var path []Addr
	for {
		if n == nil || n.Addr().Value() == nil {
			break
		}
		path = append([]Addr{n.Addr()}, path...)
		n = n.Parent()
	}
	return path
}

// walk a tree in top-down (prefix) order
func walk(tree node, path []Addr, fn func(path []Addr, n node) bool) {
	if !tree.Addr().Eq(RootAddr{}) {
		path = append(path, tree.Addr())
	}
	kontinue := fn(path, tree)
	if cmp, ok := tree.(compound); kontinue && ok {
		for _, n := range cmp.Children() {
			walk(n, path, fn)
		}
	}
}

// walk a tree in top-down (prefix) order, sorting array keys before recursing.
// more expensive
func walkSorted(tree node, path []Addr, fn func(path []Addr, n node) bool) {
	if !tree.Addr().Eq(RootAddr{}) {
		path = append(path, tree.Addr())
	}

	kontinue := fn(path, tree)
	if cmp, ok := tree.(compound); kontinue && ok {
		children := nodes(cmp.Children())
		sort.Sort(children)
		for _, n := range children {
			walkSorted(n, path, fn)
		}
	}
}


// walk a tree in bottom up (postfix) order
func walkPostfix(tree node, path []Addr, fn func(path []Addr, n node)) {
	if !tree.Addr().Eq(RootAddr{}) {
		path = append(path, tree.Addr())
	}
	if cmp, ok := tree.(compound); ok {
		for _, n := range cmp.Children() {
			walkPostfix(n, path, fn)
		}
	}
	fn(path, tree)
}

func nodeAtPath(tree node, path []Addr) (n node) {
	n = tree
	for _, addr := range path {
		if cmp, ok := n.(compound); ok {
			n = cmp.Child(addr)
			if n == nil {
				return nil
			}
		}
	}
	return
}

func addNode(tree, toAdd node, paths []Addr) {
	if cmp, ok := tree.(compound); ok && len(paths) > 0 {
		for _, addr := range paths[:len(paths)-1] {
			tree = cmp.Child(addr)
			if tree == nil {
				return
			}
		}
	}
	if cmp, ok := tree.(compound); ok {
		cmp.AddChild(toAdd)
	}
}
