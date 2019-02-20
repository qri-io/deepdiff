package difff

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
)

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

	childNames map[string]int
	children   []Node
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
	return c.children[c.childNames[name]]
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

func (d *diff) prepTrees() (t1, t2 Node, t1Nodes map[string][]Node) {
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
		t1 = tree(d.d1, "", nil, t1nodesCh)
		close(t1nodesCh)
	}()

	go func(nodes <-chan Node) {
		for range nodes {
			// do nothing
		}
		wg.Done()
	}(t2nodesCh)
	go func() {
		t2 = tree(d.d2, "", nil, t2nodesCh)
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
			name:       name,
			parent:     parent,
			childNames: map[string]int{},
			children:   make([]Node, len(x)),
			value:      v,
		}

		for i, v := range x {
			name := strconv.Itoa(i)
			node := tree(v, name, arr, nodes)
			hasher.Write(node.Hash())
			arr.childNames[name] = i
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
