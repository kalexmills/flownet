package flownet

import (
	"fmt"
	"math"
)

// Source is the ID of the source pseudonode.
const Source int = -2

// Sink is the ID of the sink pseudonode.
const Sink int = -1

// FlowNetwork is a directed graph in which each edge is associated with a capacity.
//
// By default, nodes which do not have any incoming edges are presumed to be connected to the source,
// while nodes which have no outgoing edges are presumed to be connected to the sink. These default
// source/sink connections all have maximum capacity of math.MaxInt64. The first time AddEdge is called
// with a value of either flownet.Source or flownet.Sink, all presumed edges to the respective node are
// cleared and the programmer becomes responsible for managing all edges to the respective node.
type FlowNetwork struct {
	numNodes     int
	capacity     map[edge]int64
	preflow      map[edge]int64
	excess       []int64
	label        []int
	seen         []int
	manualSource bool
	manualSink   bool
}

// Edge represents a directed edge from the node with ID 'from' to the node with ID 'to'.
type edge struct {
	from, to int
}

func (e edge) reverse() edge {
	return edge{from: e.to, to: e.from}
}

// internal source and sink IDs... why not?
const sourceID = 0
const sinkID = 1

// NewFlowNetwork constructs a new graph, preallocating enough memory for the provided number of nodes.
func NewFlowNetwork(numNodes int) FlowNetwork {
	result := FlowNetwork{
		numNodes: numNodes,
		capacity: make(map[edge]int64, 2*numNodes), // preallocate assuming avg. node degree = 2
		preflow:  make(map[edge]int64, 2*numNodes),
		excess:   make([]int64, numNodes+2),
		label:    make([]int, numNodes+2),
		seen:     make([]int, numNodes+2),
	}
	// all nodes begin their life connected to the source and sink nodes
	for i := 0; i < numNodes; i++ {
		result.capacity[edge{sourceID, i + 2}] = math.MaxInt64
		result.capacity[edge{i + 2, sinkID}] = math.MaxInt64
	}
	return result
}

// Outflow returns the amount of flow leaving the network via the sink. This is the solution to the
// typical max flow problem.
func (g FlowNetwork) Outflow() int64 {
	result := int64(0)
	for edge, flow := range g.preflow {
		if edge.to == sinkID {
			result += flow
		}
	}
	return result
}

// Flow returns the flow along an edge.
func (g FlowNetwork) Flow(from, to int) int64 {
	return g.preflow[edge{from + 2, to + 2}]
}

// Residual returns the residual flow along an edge.
func (g FlowNetwork) Residual(from, to int) int64 {
	e := edge{from + 2, to + 2}
	return g.residual(e)
}

// residual returns the same result as Residual, but could be cheaper for internal use
func (g FlowNetwork) residual(e edge) int64 {
	return g.capacity[e] - g.preflow[e]
}

// AddNode adds a new node to the graph and returns its ID.
func (g *FlowNetwork) AddNode() int {
	id := g.numNodes
	g.numNodes++
	g.excess = append(g.excess, 0)
	g.label = append(g.label, 0)
	g.capacity[edge{sourceID, id + 2}] = math.MaxInt64
	g.capacity[edge{id + 2, sinkID}] = math.MaxInt64
	return id - 2
}

// AddEdge sets the capacity of an edge in the flow network. An error is returned if either fromID or
// toID are not valid node IDs. Adding an edge twice has no additional effect. Attempting to
// use flownet.Source as toId or flownet.Sink as fromID yields an error.
func (g *FlowNetwork) AddEdge(fromID, toID int, capacity int64) error {
	if fromID < -2 || fromID >= g.numNodes {
		return fmt.Errorf("no node with ID %d is known", fromID)
	}
	if toID < -2 || toID >= g.numNodes {
		return fmt.Errorf("no node with ID %d is known", toID)
	}
	if toID == Source {
		return fmt.Errorf("no node can connect to the source pseudonode")
	}
	if fromID == Sink {
		return fmt.Errorf("no node can be connected to from the sink pseudonode")
	}
	if fromID == Source {
		g.enableManualSource()
	}
	if toID == Sink {
		g.enableManualSink()
	}

	// actually set the capacity! woo!
	g.capacity[edge{fromID + 2, toID + 2}] = capacity

	// auto-remove any connections from/to the source/sink pseudonodes (if they're managed automatically)
	if !g.manualSource {
		delete(g.capacity, edge{sourceID, toID + 2})
	}
	if !g.manualSink {
		delete(g.capacity, edge{fromID + 2, sinkID})
	}
	return nil
}

func (g *FlowNetwork) enableManualSource() {
	if g.manualSource {
		return
	}
	g.manualSource = true
	// disconnect all nodes from source/sink; programmer wants to do it themselves.
	for i := 2; i < g.numNodes+2; i++ {
		delete(g.capacity, edge{sourceID, i})
	}
}

func (g *FlowNetwork) enableManualSink() {
	if g.manualSink {
		return
	}
	g.manualSink = true
	// disconnect all nodes from source/sink; programmer wants to do it themselves.
	for i := 2; i < g.numNodes+2; i++ {
		delete(g.capacity, edge{i, sinkID})
	}
}

// PushRelabel finds a maximum flow via the push-relabel algorithm.
func (g *FlowNetwork) PushRelabel() {
	g.reset()
	// TODO: topological sort for great heuristic goodness
	nodeQueue := make([]int, 0, g.numNodes)
	for i := 0; i < g.numNodes; i++ {
		nodeQueue = append(nodeQueue, i+2)
	}
	p := len(nodeQueue) - 1
	for p >= 0 {
		u := nodeQueue[p]
		oldLabel := g.label[u]
		g.discharge(u)
		if g.label[u] > oldLabel {
			nodeQueue = append(nodeQueue[:p], nodeQueue[p+1:]...)
			nodeQueue = append(nodeQueue, u)
			p = len(nodeQueue) - 1
		} else {
			p--
		}
	}
}

// push moves all excess flow across the provided edge
func (g *FlowNetwork) push(e edge) {
	delta := min64(g.excess[e.from], g.residual(e))
	g.preflow[e] += delta
	g.preflow[e.reverse()] -= delta
	g.excess[e.from] -= delta
	g.excess[e.to] += delta
}

// relabel increases the label of an empty node to the minimum of its neighbors
func (g *FlowNetwork) relabel(nodeID int) {
	minHeight := math.MaxInt64
	for i := 0; i < g.numNodes+2; i++ {
		if g.residual(edge{nodeID, i}) > 0 {
			minHeight = min(minHeight, g.label[i])
			g.label[nodeID] = minHeight + 1
		}
	}
}

// discharge pushes as much excess from nodeID to its unvisited neighbors as possible.
func (g *FlowNetwork) discharge(nodeID int) {
	for g.excess[nodeID] > 0 {
		if g.seen[nodeID] < g.numNodes+2 {
			v := g.seen[nodeID]
			e := edge{nodeID, v}
			if g.residual(e) > 0 && g.label[nodeID] > g.label[v] {
				g.push(e)
			} else {
				g.seen[nodeID]++
			}
		} else {
			g.relabel(nodeID)
			g.seen[nodeID] = 0
		}
	}
}

// reset prepares the network for computing a new flow.
func (g *FlowNetwork) reset() {
	g.label[sourceID] = g.numNodes + 2
	g.label[sinkID] = 0
	for i := 0; i < g.numNodes; i++ {
		g.label[i+2] = 0
	}
	for id := range g.preflow {
		g.preflow[id] = 0
	}
	// set the capacity of edges from source; using the max outgoing capacity of any node adjacent to source.
	totalCapacity := int64(0) // N.B. totalCapacity only exists to force a panic on integer overflow during tests.
	for u := 0; u < g.numNodes; u++ {
		if _, ok := g.capacity[edge{sourceID, u + 2}]; !ok {
			continue
		}
		outgoingCapacity := int64(0)
		for v := 0; v < g.numNodes; v++ {
			outgoingCapacity += g.capacity[edge{u + 2, v + 2}]
		}
		g.capacity[edge{sourceID, u + 2}] = outgoingCapacity
		totalCapacity += outgoingCapacity
	}
	// saturate all outgoing edges from source by setting their excess as high as possible.
	// N.B. if the sum of the max capacity of edges leaving source exceeds math.MaxInt64, this step will
	// break and arbitrary precision arithmetic will need to be used.
	g.excess[sourceID] = math.MaxInt64
	g.push(edge{sourceID, sinkID})
	for i := 0; i < g.numNodes; i++ {
		g.push(edge{sourceID, i + 2})
	}
}

func min64(x, y int64) int64 {
	if x < y {
		return x
	}
	return y
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// SanityChecks runs several sanity checks against a FlowNetwork that has had previously had its
// flow computed.
func SanityChecks(fn FlowNetwork) error {
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
	// ensure inflow == outflow; nodeflow should be zero for every node other than source and sink.
	for node, flowDiff := range nodeflow {
		if node != sourceID && node != sinkID && flowDiff != 0 {
			return fmt.Errorf("node %d does not have its inflow equal to its outflow", node)
		}
	}
	// attempt to find an augmenting path in the graph, return an error if one is found.
	return augmentingPathCheck(fn)
}

// augmentingPathCheck returns an error if any augmenting path is found in the residual flow network.
func augmentingPathCheck(fn FlowNetwork) error {
	// run a from source to sink using the residual flow network, if you find a path, it's wrong.
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
