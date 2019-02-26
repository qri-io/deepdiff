// Package difff (with an extra f) is a structured data differ that aims for
// near-linear time complexity. It's intended to calculate differences &
// apply patches to structured data ranging from 0-500MBish of encoded JSON
//
// Diffing structured data carries additional complexity when compared to the
// standard unix diff utility, which operates on lines of text. By using the
// structure of data itself, difff is able to provide a rich description of
// changes that maps onto the structure of the data itself. difff ignores
// semantically irrelevant changes like whitespace, and can isolate changes like
// column changes to tabular data to only the relevant switches
//
// Most algorithms in this space have quadratic time complexity, which from our testing
// makes them very slow on 3MB JSON documents and unable to complete on 5MB or more.
// difff currently hovers around the 0.9Sec/MB range on 4 core processors
//
// Instead of operating on JSON directly, difff operates on document trees
// consisting of the go types created by unmarshaling from JSON, which aretwo complex types:
//   map[string]interface{}
//   []interface{}
// and five scalar types:
//   string, int, float64, bool, nil
//
// by operating on native go types difff can compare documents encoded in different
// formats, for example decoded CSV or CBOR.
//
// difff is based off an algorithm designed for diffing XML documents outlined in:
// Detecting Changes in XML Documents by Grégory Cobéna & Amélie Marian
// https://ieeexplore.ieee.org/document/994696
// it's been adapted to fit purposes of diffing for Qri: https://github.com/qri-io/qri
// the guiding use case for this work
//
// Difff also includes a tool for applying patches, see documentation for details
package difff
