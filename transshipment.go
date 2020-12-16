package flownet

import "fmt"

// A Transshipment is a circulation which allows flow to remain in a node without flowing out.
// Each node has an amount of storage, with an associated minimum and maximum.
// By default, every node in a Transshipment stores no extra flow.
//
// Transshipments can be used to model problems in which flow leaks or is consumed at certain
// points in the network.
type Transshipment struct {
	Circulation
	bounds      map[int]bounds
	specialNode int
}

type bounds struct {
	storageMax, storageMin int64
}

// NewTransshipment constructs a new graph, allocating enough capacity for the provided number of nodes.
func NewTransshipment(numNodes int) Transshipment {
	return Transshipment{
		Circulation: NewCirculation(numNodes),
		bounds:      make(map[int]bounds),
		specialNode: -1,
	}
}

// SetNodeBounds sets the upper and lower bounds on capacity which is allowed to stay in a node.
func (t *Transshipment) SetNodeBounds(nodeID int, storageMin, storageMax int64) error {
	if nodeID < 0 || t.numNodes <= nodeID {
		return fmt.Errorf("node node with ID %d is known", nodeID)
	}
	if storageMax < storageMin {
		return fmt.Errorf("storageMax cannot be smaller than storageMin: storageMin = %d, storageMax = %d", storageMin, storageMax)
	}
	t.bounds[nodeID+2] = bounds{storageMax, storageMin}
	return nil
}

// NodeFlow returns the amount of flow stored at the provided node. The results are only meaningful
// after PushRelabel has been run.
func (t *Transshipment) NodeFlow(nodeID int) int64 {
	return t.Circulation.Flow(nodeID, t.specialNode)
}

// PushRelabel finds a valid transshipment (if one exists) via the push-relabel algorithm.
func (t *Transshipment) PushRelabel() {
	// N.B. a transshipment can be obtained from a circulation by adding fake edges
	// to a new node that can store any flow that ends up being 'stored' at the nodes.
	if t.specialNode == -1 {
		t.specialNode = t.Circulation.AddNode()
	}
	for nodeID, bounds := range t.bounds {
		t.Circulation.AddEdge(nodeID, t.specialNode, bounds.storageMax, bounds.storageMin)
	}
	t.Circulation.PushRelabel()
}
