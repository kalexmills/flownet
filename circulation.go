// Package flownet provides algorithms for solving optimization problems on a flow network.
package flownet

import (
	"fmt"
	"math"
)

// TODO: we have to use multiple sources and sinks to handle BOTH node demands AND edge demands!
//       whenever we add a new constraint, we connect the old source/sink to one another with an infinite capacity edge
//       then we use the new source/sink to enforce the new constraints.

// A Circulation is a flow network which has an additional demand associated with each of its nodes
// or edges. Flow may be supplied to the network via negative node demands.
//
// Whereas in a traditional flow network problem we are interested in maximizing the amount of flow
// from the source to the sink, in a circulation we ask if there is a feasible flow which satisfies
// the demand. Nodes in a circulation are not connected the source or sink as in a traditional flow
// network. Trying to add these connections to a Circulation will cause an error.
type Circulation struct {
	FlowNetwork
	// demand stores the demand for each edge
	demand map[edge]int64
	// nodeDemand stores the demand for each node
	nodeDemand map[int]int64
	// special source node used for node demands
	nodeSource int
	// special sink node used for node demands
	nodeSink int
	// amount of flow expected in a valid circulation.
	targetValue int64
}

// NewCirculation constructs a new graph allocating initial capacity for the provided number of nodes.
func NewCirculation(numNodes int) Circulation {
	return Circulation{
		FlowNetwork: NewFlowNetwork(numNodes),
		demand:      make(map[edge]int64),
		nodeDemand:  make(map[int]int64),
	}
}

// SetNodeDemand sets the demand for a node.
func (c *Circulation) SetNodeDemand(nodeID int, demand int64) error {
	if nodeID == Source || nodeID == Sink {
		return fmt.Errorf("no demand can be set for the source or sink")
	}
	if nodeID < 0 || nodeID >= c.numNodes {
		return fmt.Errorf("no node with id %d is known", nodeID)
	}
	if demand != 0 && c.nodeSource == 0 {
		c.nodeSource = c.AddNode()
		c.nodeSink = c.AddNode()
		c.FlowNetwork.AddEdge(c.nodeSink, c.nodeSource, math.MaxInt64)
	}
	if demand == 0 {
		c.FlowNetwork.AddEdge(c.nodeSource, nodeID, 0)
		c.FlowNetwork.AddEdge(nodeID, c.nodeSink, 0)
	}
	if demand > 0 {
		c.FlowNetwork.AddEdge(nodeID, c.nodeSink, demand)
	}
	if demand < 0 {
		c.FlowNetwork.AddEdge(c.nodeSource, nodeID, -demand)
	}
	c.nodeDemand[nodeID+2] = demand
	return nil
}

// AddEdge sets the capacity and demand of the edge in the flow network. An error is returned
// if either fromID or toID are not valid node IDs. Adding an edge twice has no additional effect.
// Setting demands on edges also updates the demand on the adjacent nodes.
func (c *Circulation) AddEdge(fromID, toID int, capacity, demand int64) error {
	if fromID == Source || fromID == Sink || toID == Source || toID == Sink {
		return fmt.Errorf("edges to/from the source/sink nodes cannot be used in a Circulation")
	}
	if capacity < demand {
		return fmt.Errorf("capacity cannot be smaller than demand; capacity = %d, demand = %d", capacity, demand)
	}
	if err := c.FlowNetwork.AddEdge(fromID, toID, capacity); err != nil {
		return err
	}
	e := edge{fromID + 2, toID + 2}

	c.capacity[e] = capacity - demand

	if demand != 0 {
		c.demand[e] = demand
	}
	if demand == 0 {
		delete(c.demand, e)
	}
	return nil
}

// Capacity returns the capacity of the provided edge.
func (c *Circulation) Capacity(from, to int) int64 {
	return c.FlowNetwork.Capacity(from, to) + c.demand[edge{from + 2, to + 2}]
}

// Flow returns the flow achieved by the circulation along the provided edge. The results are
// only meaningful after PushRelabel has been run.
func (c *Circulation) Flow(from, to int) int64 {
	return c.FlowNetwork.Flow(from, to) + c.demand[edge{from + 2, to + 2}]
}

// EdgeDemand returns the demand required along each edge.
func (c *Circulation) EdgeDemand(from, to int) int64 {
	return c.demand[edge{from + 2, to + 2}]
}

// NodeDemand returns the demand required at each node.
func (c *Circulation) NodeDemand(nodeID int) int64 {
	return c.nodeDemand[nodeID+2]
}

// SatisfiesDemand is true iff the flow satisfies all required demand.
func (c *Circulation) SatisfiesDemand() bool {
	return c.Outflow() == c.targetValue
}

// PushRelabel finds a valid circulation (if one exists) via the push-relabel algorithm.
func (c *Circulation) PushRelabel() {
	if len(c.demand) == 0 && len(c.nodeDemand) == 0 {
		c.FlowNetwork.PushRelabel()
		return
	}
	// disconnect the source and sink nodes; they don't work the same for circulations with demands
	for edge := range c.FlowNetwork.capacity {
		if edge.from == sourceID {
			delete(c.FlowNetwork.capacity, edge)
		}
		if edge.to == sinkID {
			delete(c.FlowNetwork.capacity, edge)
		}
	}
	targetValue := int64(0)
	for e, demand := range c.demand {
		if demand != 0 {
			c.FlowNetwork.capacity[edge{e.from, sinkID}] += demand
			c.FlowNetwork.adjacencyList[e.from][sinkID] = struct{}{}

			c.FlowNetwork.capacity[edge{sourceID, e.to}] += demand
			c.FlowNetwork.adjacencyList[sourceID][e.to] = struct{}{}
		}
		if demand > 0 {
			targetValue += demand
		}
	}

	if len(c.demand) == 0 { // handle no edge demands
		c.FlowNetwork.capacity[edge{sourceID, c.nodeSource + 2}] = math.MaxInt64
		c.FlowNetwork.capacity[edge{c.nodeSink + 2, sinkID}] = math.MaxInt64
		delete(c.FlowNetwork.capacity, edge{c.nodeSink + 2, c.nodeSource + 2})
		for _, demand := range c.nodeDemand {
			if demand > 0 {
				targetValue += demand
			}
		}
	}
	c.targetValue = targetValue

	// find the max-flow in the resulting flow network.
	c.FlowNetwork.PushRelabel()
}
