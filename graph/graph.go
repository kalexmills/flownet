package graph

import (
	"fmt"
	"math"
)

type Graph struct {
	numNodes int
	capacity map[edge]int64
	preflow  map[edge]int64
	excess   []int64
	label    []int
	seen     []int
}

// Edge represents a directed edge from the node with ID 'from' to the node with ID 'to'.
type edge struct {
	from, to int
}

func (e edge) reverse() edge {
	return edge{from: e.to, to: e.from}
}

const sourceID = 0
const sinkID = 1

// NewGraph constructs a new graph, allocating an initial capacity for the provided number of nodes.
func NewGraph(numNodes int) Graph {
	result := Graph{
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

// Outflow returns the amount of flow leaving the network via the sink.
func (g Graph) Outflow() int64 {
	result := int64(0)
	for edge, flow := range g.preflow {
		if edge.to == sinkID {
			result += flow
		}
	}
	return result
}

// Flow returns the flow along an edge.
func (g Graph) Flow(from, to int) int64 {
	return g.preflow[edge{from + 2, to + 2}]
}

// Residual returns the residual flow along an edge.
func (g Graph) Residual(from, to int) int64 {
	e := edge{from + 2, to + 2}
	return g.capacity[e] - g.preflow[e]
}

// residual returns the same result as Residual, but could be cheaper for internal use
func (g Graph) residual(e edge) int64 {
	return g.capacity[e] - g.preflow[e]
}

// AddNode adds a new node to the graph and returns its ID.
func (g *Graph) AddNode() int {
	id := g.numNodes
	g.numNodes++
	g.excess = append(g.excess, 0)
	g.label = append(g.label, 0)
	g.capacity[edge{sourceID, id + 2}] = math.MaxInt64
	g.capacity[edge{id + 2, sinkID}] = math.MaxInt64
	return id - 2
}

// AddEdge sets the capacity of an edge in the flow network. An error is returned if either fromID or
// toID are not valid node IDs.
func (g *Graph) AddEdge(fromID, toID int, capacity int64) error {
	if fromID < 0 || fromID >= g.numNodes {
		return fmt.Errorf("no node with ID %d is known", fromID)
	}
	if toID < 0 || toID >= g.numNodes {
		return fmt.Errorf("no node with ID %d is known", toID)
	}
	g.capacity[edge{fromID + 2, toID + 2}] = capacity
	// remove any connections from/to the source/sink pseudonodes, if they exist.
	delete(g.capacity, edge{sourceID, toID + 2})
	delete(g.capacity, edge{fromID + 2, sinkID})
	return nil
}

// PushRelabel finds a maximum flow via the push-relabel algorithm.
func (g *Graph) PushRelabel() {
	g.reset()
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

func (g *Graph) active(nodeID int) bool {
	return nodeID != sinkID && g.excess[nodeID] > 0
}

// push moves all excess flow across the provided edge
func (g *Graph) push(e edge) {
	delta := min64(g.excess[e.from], g.residual(e))
	fmt.Printf("push    %d units from %d -> %d\n", delta, e.from-2, e.to-2)
	g.preflow[e] += delta
	g.preflow[e.reverse()] -= delta
	g.excess[e.from] -= delta
	g.excess[e.to] += delta
}

// relabel increases the label of an empty node to the minimum of its neighbors
func (g *Graph) relabel(nodeID int) {
	priorLabel := g.label[nodeID]
	minHeight := math.MaxInt64
	for i := 0; i < g.numNodes+2; i++ {
		if g.residual(edge{nodeID, i}) > 0 {
			minHeight = min(minHeight, g.label[i])
			g.label[nodeID] = minHeight + 1
		}
	}
	fmt.Printf("relabel %d from %d to %d\n", nodeID-2, priorLabel, g.label[nodeID])
}

// discharge pushes as much excess from nodeID to its unvisited neighbors as possible.
func (g *Graph) discharge(nodeID int) {
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
func (g *Graph) reset() {
	g.label[sourceID] = g.numNodes + 2
	g.label[sinkID] = 0
	for i := 0; i < g.numNodes; i++ {
		g.label[i+2] = 0
	}
	for id := range g.preflow {
		g.preflow[id] = 0
	}
	// set the capacity of edges from source; using the max outgoing capacity of any node adjacent to source.
	totalCapacity := int64(0) // N.B. totalCapacity exists to force a panic on integer overflow during tests.
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
