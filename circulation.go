package flownet

import "fmt"

// Circulation is a flow network which additionally requires every edge in the flow network to satisfy
// a certain amount of demand.
// Whereas in a traditional flow network problem we are interested in maximizing the amount of flow
// from the source to the sink, in a circulation we ask if there is a feasible flow which satisfies
// the demand.Nodes in a circulation are not connected the source or sink as in a
// traditional flow network, trying to add these connections to a Circulation will result in an error.
type Circulation struct {
	FlowNetwork
	demand map[edge]int64
	// targetValue is only reached when all lower-bounds are satisfied
	targetValue int64
}

// NewCirculation constructs a new graph allocating initial capacity for the provided number of nodes.
func NewCirculation(numNodes int) Circulation {
	return Circulation{
		FlowNetwork: NewFlowNetwork(numNodes),
		demand:      make(map[edge]int64),
	}
}

// AddEdge sets the capacity and demand of the edge in the flow network. An error is returned
// if either fromID or toID are not valid node IDs. Adding an edge twice has no additional effect.
func (c *Circulation) AddEdge(fromID, toID int, capacity, demand int64) error {
	if fromID == Source || fromID == Sink || toID == Source || toID == Sink {
		return fmt.Errorf("edges to/from the source/sink nodes cannot be used in a Circulation")
	}
	if err := c.FlowNetwork.AddEdge(fromID, toID, capacity); err != nil {
		return err
	}
	if capacity < demand {
		return fmt.Errorf("capacity cannot be smaller than demand; capacity = %d, demand = %d", capacity, demand)
	}
	c.demand[edge{fromID + 2, toID + 2}] = demand
	return nil
}

// Flow returns the flow achieved by the circulation along the provided edge. The results are
// only meaningful after PushRelabel has been run.
func (c *Circulation) Flow(from, to int) int64 {
	return c.FlowNetwork.Flow(from, to) + c.demand[edge{from, to}]
}

// SatisfiesDemand is true iff the flow satisfies all required demand.
func (c *Circulation) SatisfiesDemand() bool {
	return c.FlowNetwork.Outflow() == c.targetValue
}

// PushRelabel finds a valid circulation (if one exists) via the push-relabel algorithm.
func (c *Circulation) PushRelabel() {
	// disconnect the source and sink nodes; they don't work the same for circulations.
	for edge := range c.FlowNetwork.capacity {
		if edge.from == sourceID {
			delete(c.FlowNetwork.capacity, edge)
		}
		if edge.to == sinkID {
			delete(c.FlowNetwork.capacity, edge)
		}
	}
	// compute the excess demand at each node
	excessDemand := make([]int64, c.FlowNetwork.numNodes)
	for edge, demand := range c.demand {
		c.FlowNetwork.capacity[edge] -= demand
		excessDemand[edge.from] -= demand
		excessDemand[edge.to] += demand
	}
	// set the capacities on the source and sink nodes according to excess demand
	targetValue := int64(0)
	for u, excessD := range excessDemand {
		if excessD > 0 {
			c.FlowNetwork.capacity[edge{sourceID, u}] = excessD
			targetValue += excessD
		}
		if excessD < 0 {
			c.FlowNetwork.capacity[edge{u, sinkID}] = -excessD
		}
	}
	c.targetValue = targetValue
	// find the max-flow in the resulting flow network.
	c.FlowNetwork.PushRelabel()
}

// SanityCheckCirculation runs sanity checks against a circulation that has previously had its flow computed.
func SanityCheckCirculation(c Circulation) error {
	err := SanityCheckFlowNetwork(c.FlowNetwork)
	if err != nil {
		return err
	}
	if !c.SatisfiesDemand() {
		// we have nothing  to check unless demand was satisfied
		return nil
	}
	for edge, flow := range c.FlowNetwork.preflow {
		if flow > 0 {
			if flow < c.demand[edge] {
				return fmt.Errorf("edge from %d to %d has flow %d which is less than demand %d", edge.from, edge.to, flow, c.demand[edge])
			}
		}
	}
	return nil
}
