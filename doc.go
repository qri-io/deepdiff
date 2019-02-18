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
//
// TODO (b5):
//  √ basic non-move-delta tests passing
//  √ adjust dst paths after insert/delete calculations
//  * move deltas:
//    * parent-switch moves
//  	* longest-common-match subsequence
//  	* internal node move deltas
//  * initial tests passing
//  * patch application
//  * basic test that checks forward patch against d2
//  * basic test the checks backward patch against d1 (confirm patch is reversible)
//  * rename "Compound" -> "Internal", "Scalar" -> "Leaf", deal with possible empty object/array bugs
//  --
//  * dataset generator, benchmarks against 100MB & 500MB datasets
//  * change simulator & generative tests
//  * fast-path using simplified cubic-time diff calculation
//  * write example
package difff
