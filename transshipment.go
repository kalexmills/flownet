package flownet

import "fmt"

// Transshipment is a circulation which does not require that the amount of flow entering a node
// remains strictly equal to the amount of flow exiting a node. In a transshipment, some of the
// flow is allowed to stay pooled up in the node. Each node also has a capacity and a demand. By
// default, every node has zero capacity and demand.
type Transshipment struct {
	Circulation
	bounds      map[int]bounds
	specialNode int // a special node used to model node demand/capacity
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

type bounds struct {
	capacity, demand int64
}

// SanityCheckTransshipment runs sanity checks and reports them as appropriate for a Transshipment.
func SanityCheckTransshipment(t Transshipment) error {
	err := SanityCheckFlowNetwork(t.FlowNetwork)
	if err != nil {
		return err
	}
	for nodeID, bounds := range t.bounds {
		if bounds.capacity < t.NodeFlow(nodeID) {
			return fmt.Errorf("node %d has stored flow of %d which exceeds its capacity bound of %d", nodeID, t.NodeFlow(nodeID), bounds.capacity)
		}
	}
	if !t.SatisfiesDemand() {
		return nil
	}
	for nodeID, bounds := range t.bounds {
		if t.NodeFlow(nodeID) < bounds.demand {
			return fmt.Errorf("node %d has stored flow of %d which does not meet or exceed its demand of %d", nodeID, t.NodeFlow(nodeID), bounds.demand)
		}
	}
	return SanityCheckCirculation(t.Circulation)
}
