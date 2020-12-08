package flownet

import "fmt"

// A Transshipment is a circulation which does not require that the amount of flow entering a node
// remains strictly equal to the amount of flow exiting a node. In a transshipment, some of the
// flow is allowed to stay pooled up in the node. Each node also has a capacity and a demand. By
// default, every node has zero capacity and demand.
type Transshipment struct {
	Circulation
	bounds      map[int]bounds
	specialNode int // a special node used to model node demand/capacity
}

type bounds struct {
	capacity, demand int64
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
func (t *Transshipment) SetNodeBounds(nodeID int, capacity, demand int64) error {
	if nodeID < 0 || t.numNodes <= nodeID {
		return fmt.Errorf("node node with ID %d is known", nodeID)
	}
	if capacity < demand {
		return fmt.Errorf("capacity cannot be smaller than demand: capacity = %d, demand = %d", capacity, demand)
	}
	t.bounds[nodeID+2] = bounds{capacity, demand}
	return nil
}

// NodeFlow returns the amount of flow stored at the provided node. The results are only meaningful
// after PushRelabel has been run.
func (t *Transshipment) NodeFlow(nodeID int) int64 {
	return t.Circulation.Flow(nodeID, t.specialNode)
}

// PushRelabel finds a valid transshipment (if one exists) via the push-relabel algorithm.
func (t *Transshipment) PushRelabel() {
	// N.B. a transshipment can be obtained from a circulation by cheating. We add fake edges
	// that store any flow that ends up being 'stored' at the nodes.
	if t.specialNode == -1 {
		t.specialNode = t.Circulation.AddNode()
	}
	for nodeID, bounds := range t.bounds {
		t.Circulation.AddEdge(nodeID, t.specialNode, bounds.capacity, bounds.demand)
	}
	t.Circulation.PushRelabel()
}
