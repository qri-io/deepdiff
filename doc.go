// Package difff calculates the differences of document trees consisting of the
// go types created by unmarshaling from JSON, consisting of two complex types:
//   map[string]interface{}
//   []interface{}
// and five scalar types:
//   string, int, float64, bool, nil
//
// difff is based off an algorithm designed for diffing XML documents outlined in:
//    Detecting Changes in XML Documents by Grégory Cobéna & Amélie Marian
//
// The paper describes an algorithm for generating an edit script that transitions
// between two states of tree-type data structures (XML)
package difff
