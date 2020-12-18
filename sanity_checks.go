package flownet

import "fmt"

// SanityChecks contains sanity check procedures for FlowNetworks, Transshipments, and Circulations.
var SanityChecks SanityCheckers

// SanityCheckers holds sanity check procedures for flownet types.
type SanityCheckers struct{}

// FlowNetwork runs several sanity checks against a FlowNetwork that has had previously had its
// flow computed. If flowEquality is true, the inflow of each node is checked to ensure it is equal
// to the outflow.
func (sc SanityCheckers) FlowNetwork(fn FlowNetwork, flowEquality bool) error {
	nodeflow := make(map[int]int64) // computes residual flow stored at nodes to ensure inflow == outflow
	for e, flow := range fn.preflow {
		if cap, ok := fn.capacity[e]; ok {
			if flow > cap {
				return fmt.Errorf("capacity of %d on edge from %d to %d exceeded by flow %d", cap, e.from, e.to, flow)
			}
			nodeflow[e.from] -= flow
			nodeflow[e.to] += flow
		} else {
			if _, ok := fn.capacity[edge{e.to, e.from}]; flow > 0 || (flow < 0 && !ok) {
				return fmt.Errorf("flow of %d reported on edge from %d to %d, but found no capacity record for that edge", flow, e.from, e.to)
			}
		}
	}
	// ensure inflow == outflow; node flow should be zero for every node other than source and sink.
	if flowEquality {
		for node, flowDiff := range nodeflow {
			if node != sourceID && node != sinkID && flowDiff != 0 {
				return fmt.Errorf("node %d does not have its inflow equal to its outflow", node)
			}
		}
	}
	// attempt to find an augmenting path in the graph return an error if one is found.
	return sc.augmentingPathCheck(fn)
}

// augmentingPathCheck returns an error if any augmenting path is found in the residual flow network.
func (SanityCheckers) augmentingPathCheck(fn FlowNetwork) error {
	// run a BFS from source to sink using the residual flow network, if you find a path, it's wrong.
	frontier := []int{sourceID}
	visited := make(map[int]struct{})
	for len(frontier) > 0 {
		curr := frontier[0]
		frontier = frontier[1:]
		visited[curr] = struct{}{}
		for i := 0; i < fn.numNodes+2; i++ {
			if _, ok := visited[i]; !ok && fn.residual(edge{curr, i}) > 0 {
				if i == sinkID {
					return fmt.Errorf("found an augmenting path from source to sink via edge %d; flow is not maximum", curr)
				}
				frontier = append(frontier, i)
			}
		}
	}
	return nil
}

// Circulation runs sanity checks against a circulation that has previously had its flow computed. These
// sanity checks include the FlowNetwork checks; they do not need to be run separately.
func (sc SanityCheckers) Circulation(c Circulation) error {
	err := sc.FlowNetwork(c.FlowNetwork, true)
	if err != nil {
		return err
	}
	if !c.SatisfiesDemand() {
		// we have nothing to check unless demand was satisfied
		return nil
	}
	for edge := range c.FlowNetwork.preflow {
		flow := c.Flow(edge.from, edge.to)
		demand := c.EdgeDemand(edge.from, edge.to)
		if flow < demand {
			return fmt.Errorf("edge from %d to %d has flow %d which is less than demand %d", edge.from, edge.to, flow, demand)
		}
	}
	// TODO: check node demand
	return nil
}

// Transshipment runs sanity checks and reports them as appropriate for a Transshipment. These sanity
// checks include the Circulation and FlowNetwork checks; they do not need to be run separately.
func (sc SanityCheckers) Transshipment(t Transshipment) error {
	err := sc.FlowNetwork(t.FlowNetwork, false)
	if err != nil {
		return err
	}
	for nodeID, bounds := range t.bounds {
		if bounds.storageMax < t.NodeFlow(nodeID) {
			return fmt.Errorf("node %d has stored flow of %d which exceeds its capacity bound of %d", nodeID, t.NodeFlow(nodeID), bounds.storageMax)
		}
	}
	if !t.SatisfiesDemand() {
		return nil
	}
	for nodeID, bounds := range t.bounds {
		if t.NodeFlow(nodeID) < bounds.storageMin {
			return fmt.Errorf("node %d has stored flow of %d which does not meet or exceed its demand of %d", nodeID, t.NodeFlow(nodeID), bounds.storageMin)
		}
	}
	return sc.Circulation(t.Circulation)
}
