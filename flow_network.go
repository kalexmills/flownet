package flownet

import (
	"container/heap"
	"fmt"
	"log"
	"math"
)

// Source is the ID of the source pseudonode.
const Source int = -2

// Sink is the ID of the sink pseudonode.
const Sink int = -1

// A FlowNetwork is a directed graph which can be used to solve maximum-flow problems. Each edge is
// associated with a capacity and a flow. The flow on each edge may not exceed the stated capacity.
// Each node may be connected to a source or a sink node.
//
// By default, nodes which do not have any incoming edges are presumed to be connected to the source,
// while nodes which have no outgoing edges are presumed to be connected to the sink. These default
// source/sink connections all have maximum capacity of math.MaxInt64. The first time AddEdge is called
// with a value of either flownet.Source or flownet.Sink, all the presumptive edges to the respective
// node are cleared and the programmer becomes responsible for managing all edges to the Source or Sink,
// respectively.
type FlowNetwork struct {
	// numNodes is the total number of nodes in this network other than the source and sink.
	numNodes int
	// nodeOrder contains the order in which nodes are discharged.
	nodeOrder []int
	// adjacencyList is a map from source nodes to a set of destination nodes in no particular order.
	adjacencyList []map[int]struct{}
	// adjacencyVisitList is a list of adjacency lists in the order nodes are visited.
	adjacencyVisitList [][]int
	// capacity contains a map from each edge to its capacity.
	capacity map[edge]int64
	// preflow contains a map from each edge to its flow value.
	preflow map[edge]int64
	// excess stores the excess flow at each node.
	excess []int64
	// label stores the label of each node.
	label []int
	// seen stores the last node seen by each node for use during the discharge operation.
	seen []int
	// manualSource is true only if the programmer has manually added an edge leaving flownet.Source.
	manualSource bool
	// manualSink is true only if the programmer has manually added an edge entering flownet.Sink.
	manualSink bool
}

// Edge represents a directed edge from the node with ID 'from' to the node with ID 'to'.
type edge struct {
	from, to int
}

// reverse returns the reversed edge.
func (e edge) reverse() edge {
	return edge{from: e.to, to: e.from}
}

// sourceID is the internal id for the source node.
const sourceID = 0

// sinkID is the internal id for the sink node.
const sinkID = 1

// NewFlowNetwork constructs a new graph, preallocating enough memory for the provided number of nodes.
func NewFlowNetwork(numNodes int) FlowNetwork {
	result := FlowNetwork{
		numNodes:      numNodes,
		adjacencyList: make([]map[int]struct{}, numNodes+2),
		capacity:      make(map[edge]int64, 2*numNodes), // preallocate assuming avg. node degree = 2
		preflow:       make(map[edge]int64, 2*numNodes),
		excess:        make([]int64, numNodes+2),
		label:         make([]int, numNodes+2),
		seen:          make([]int, numNodes+2),
	}
	result.adjacencyList[sourceID] = make(map[int]struct{})
	result.adjacencyList[sinkID] = make(map[int]struct{})
	// all nodes begin their life connected to the source and sink nodes
	for i := 0; i < numNodes; i++ {
		result.adjacencyList[i+2] = make(map[int]struct{})

		result.adjacencyList[sourceID][i+2] = struct{}{}
		result.adjacencyList[i+2][sinkID] = struct{}{}

		result.capacity[edge{sourceID, i + 2}] = math.MaxInt64
		result.capacity[edge{i + 2, sinkID}] = math.MaxInt64
	}
	return result
}

// Outflow returns the amount of flow which leaves the network via the sink. After PushRelabel has
// been called, this will be a solution to the max-flow problem.
func (g FlowNetwork) Outflow() int64 {
	result := int64(0)
	for edge, flow := range g.preflow { // TODO: optimize via caching
		if edge.to == sinkID {
			result += flow
		}
	}
	return result
}

// Flow returns the flow along an edge. Before PushRelabel is called this method returns 0.
func (g FlowNetwork) Flow(from, to int) int64 {
	return g.preflow[edge{from + 2, to + 2}]
}

// Residual returns the residual flow along an edge, defined as capacity - flow.
func (g FlowNetwork) Residual(from, to int) int64 {
	return g.residual(edge{from + 2, to + 2})
}

// Capacity returns the capacity of the provided edge.
func (g FlowNetwork) Capacity(from, to int) int64 {
	return g.capacity[edge{from + 2, to + 2}]
}

// residual returns the same result as Residual, but could be cheaper for internal use.
func (g FlowNetwork) residual(e edge) int64 {
	if g.capacity[e] == 0 {
		return g.preflow[e.reverse()]
	}
	return g.capacity[e] - g.preflow[e]
}

// AddNode adds a new node to the graph and returns its ID, which must be used in subsequent
// calls.
func (g *FlowNetwork) AddNode() int {
	id := g.numNodes
	g.numNodes++
	g.excess = append(g.excess, 0)
	g.label = append(g.label, 0)
	g.seen = append(g.seen, 0)
	g.adjacencyList = append(g.adjacencyList, make(map[int]struct{}))
	if !g.manualSource {
		g.capacity[edge{sourceID, id + 2}] = math.MaxInt64
		g.adjacencyList[sourceID][id+2] = struct{}{}
	}
	if !g.manualSink {
		g.capacity[edge{id + 2, sinkID}] = math.MaxInt64
		g.adjacencyList[id+2][sinkID] = struct{}{}
	}
	return id
}

// AddEdge sets the capacity of an edge in the flow network. Adding an edge twice has no additional effect.
// Attempting to use flownet.Source as toId or flownet.Sink as fromID yields an error. An error is returned
// if either fromID or toID are not valid node IDs.
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

	// actually set the capacity! woo! (finally)
	g.capacity[edge{fromID + 2, toID + 2}] = capacity
	g.adjacencyList[fromID+2][toID+2] = struct{}{}

	// auto-remove any connections from/to the source/sink pseudonodes (if they're managed automatically)
	if !g.manualSource {
		delete(g.capacity, edge{sourceID, toID + 2})
		delete(g.adjacencyList[sourceID], toID+2)
	}
	if !g.manualSink {
		delete(g.capacity, edge{fromID + 2, sinkID})
		delete(g.adjacencyList[fromID+2], sinkID)
	}
	return nil
}

// SetNodeOrder sets the order in which nodes are initially visited by the PushRelabel algorithm. By default, nodes
// are first visited in order of ID, then in descending order of label. As long as all of the nodeIDs are
// contained in the provided array, the PushRelabel algorithm will work properly. If some nodeID is missing, an error
// is returned and the order will remain unchanged. If any node is added after SetNodeOrder is called, the node order
// will reset to the default.
//
// The node order set here only affects the initial node ordering for the purposes of the push-relabel
// algorithm. Any relabeling that occurs during the algorithm may alter this order in unintuitive ways.
func (g *FlowNetwork) SetNodeOrder(nodeIDs []int) error {
	if len(nodeIDs) != g.numNodes {
		return fmt.Errorf("wrong number of nodeIDs; expected exactly %d of them", g.numNodes)
	}
	ids := make(map[int]struct{})
	mappedIds := make([]int, g.numNodes)
	// reverse the nodeIDs here, since PushRelabel's queue runs backwards
	for i, id := range nodeIDs {
		if id < 0 || id >= g.numNodes {
			return fmt.Errorf("unknown node ID %d", id)
		}
		ids[id] = struct{}{}
		mappedIds[g.numNodes-1-i] = id + 2
	}
	if len(ids) != g.numNodes {
		return fmt.Errorf("duplicate nodeIDs were present, saw %d unique ids", len(ids))
	}
	g.nodeOrder = mappedIds
	return nil
}

// PushRelabel finds a maximum flow via the relabel-to-front variant of the push-relabel algorithm. More
// specifically, PushRelabel visits each node in the network in the node order and attempts to discharges
// excess flow from the node. This may update the node's label. When a node's label changes as a result of
// the algorithm, it is moved to the front of the node order, and all nodes are visited once more.
func (g *FlowNetwork) PushRelabel() {
	if len(g.nodeOrder) != g.numNodes {
		g.nodeOrder = make([]int, 0, g.numNodes)
		for i := 0; i < g.numNodes; i++ {
			g.nodeOrder = append(g.nodeOrder, g.numNodes-1-i+2)
		}
	}
	g.reset()
	nodeQueue := append(make([]int, 0, g.numNodes), g.nodeOrder...)
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

// push moves as much excess flow across the provided edge as possible without violating the edge's capacity
// constraint.
func (g *FlowNetwork) push(e edge) {
	delta := min64(g.excess[e.from], g.residual(e))
	if g.capacity[e] > 0 {
		g.preflow[e] += delta
	} else {
		g.preflow[e.reverse()] -= delta
	}
	g.excess[e.from] -= delta
	g.excess[e.to] += delta
}

// relabel increases the label of an node with no excess to one larger than the minimum of its neighbors.
func (g *FlowNetwork) relabel(nodeID int) {
	minHeight := math.MaxInt32 - 1
	for _, u := range g.adjacencyVisitList[nodeID] {
		if g.residual(edge{nodeID, u}) > 0 {
			minHeight = min(minHeight, g.label[u])
			g.label[nodeID] = minHeight + 1
		}
	}
	if minHeight+1 == math.MaxInt32 {
		log.Fatalf("could not relabel node %d", nodeID-2)
	}
}

// discharge pushes as much excess from nodeID to its unseen neighbors as possible.
func (g *FlowNetwork) discharge(nodeID int) {
	for g.excess[nodeID] > 0 {
		if g.seen[nodeID] == len(g.adjacencyVisitList[nodeID]) {
			g.relabel(nodeID)
			g.seen[nodeID] = 0
		} else {
			v := g.adjacencyVisitList[nodeID][g.seen[nodeID]]
			e := edge{nodeID, v}
			if g.residual(e) > 0 && g.label[nodeID] == g.label[v]+1 {
				g.push(e)
			} else {
				g.seen[nodeID]++
			}
		}
	}
}

// reset prepares the network for computing a new flow.
func (g *FlowNetwork) reset() {
	// construct an adjacency visit list that is compatible with nodeOrder (since nodeOrder may have changed.)
	// TODO: we don't need to do this if the nodeOrder or set of nodes _hasn't_ changed.
	g.adjacencyVisitList = make([][]int, len(g.adjacencyList))
	for u := range g.adjacencyList {
		for _, v := range append(g.nodeOrder, []int{sourceID, sinkID}...) {
			_, ok1 := g.adjacencyList[u][v]
			_, ok2 := g.adjacencyList[v][u]
			if ok1 || ok2 {
				g.adjacencyVisitList[u] = append([]int{v}, g.adjacencyVisitList[u]...)
			}
		}
	}
	g.label[sourceID] = g.numNodes + 2
	g.label[sinkID] = 0
	for i := 0; i < g.numNodes; i++ {
		g.label[i+2] = 0
	}
	for e := range g.preflow {
		g.preflow[e] = 0
	}
	// set the capacity, excess, and flow for edges leading out from from source; using the max outgoing capacity of any node adjacent to source.
	totalCapacity := int64(0)
	for u := 0; u < g.numNodes; u++ {
		if _, ok := g.capacity[edge{sourceID, u + 2}]; !ok {
			continue
		}
		outgoingCapacity := int64(0)
		for v := range g.adjacencyList[u+2] {
			if v == sinkID || v == sourceID {
				continue
			}
			outgoingCapacity += g.capacity[edge{u + 2, v}]
		}
		totalCapacity += outgoingCapacity

		g.capacity[edge{sourceID, u + 2}] = outgoingCapacity
		g.excess[u+2] = outgoingCapacity
		g.preflow[edge{sourceID, u + 2}] = outgoingCapacity
	}
	g.excess[sourceID] = -totalCapacity
}

func (g *FlowNetwork) enableManualSource() {
	if g.manualSource {
		return
	}
	g.manualSource = true
	// disconnect all nodes from source/sink; programmer wants to do it themselves.
	for i := 2; i < g.numNodes+2; i++ {
		delete(g.capacity, edge{sourceID, i})
		delete(g.adjacencyList[sourceID], i)
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
		delete(g.adjacencyList[i], sinkID)
	}
}

// TopSort returns a topological ordering of the nodes in the provided FlowNetwork, starting from the
// nodes connected to the source, using the provided less function to break any ties that are found.
// if the flow network is not a DAG (which is allowed) this function reports an error.
func TopSort(fn FlowNetwork, less func(int, int) bool) ([]int, error) {
	unvisitedEdges := make([]map[int]struct{}, fn.numNodes+2) // list of nodeIDs to the set of their of incoming nodes
	for edge, capacity := range fn.capacity {
		if capacity <= 0 {
			continue
		}
		if unvisitedEdges[edge.to] == nil {
			unvisitedEdges[edge.to] = make(map[int]struct{})
		}
		unvisitedEdges[edge.to][edge.from] = struct{}{}
	}
	roots := &nodeHeap{ // stores all nodes with no incoming edge, sorted in order of less
		nodeIDs: []int{sourceID},
		less:    less,
	}
	heap.Init(roots)
	result := make([]int, 0, fn.numNodes)
	for roots.Len() > 0 {
		next := roots.Pop().(int)
		if next != sourceID && next != sinkID {
			result = append(result, next-2)
		}
		for neighbor := range fn.adjacencyList[next] {
			delete(unvisitedEdges[neighbor], next)
			if len(unvisitedEdges[neighbor]) == 0 {
				heap.Push(roots, neighbor)
			}
		}
	}
	leftoverEdges := 0
	for _, edges := range unvisitedEdges {
		leftoverEdges += len(edges)
	}
	if leftoverEdges > 0 {
		return nil, fmt.Errorf("graph has a cycle")
	}
	return result, nil
}

// nodeHeap stores a heap of nodeIDs sorted by the provided less function.
type nodeHeap struct {
	nodeIDs []int
	less    func(int, int) bool
}

func (h nodeHeap) Len() int           { return len(h.nodeIDs) }
func (h nodeHeap) Less(i, j int) bool { return h.less(h.nodeIDs[i], h.nodeIDs[j]) }
func (h nodeHeap) Swap(i, j int)      { h.nodeIDs[i], h.nodeIDs[j] = h.nodeIDs[j], h.nodeIDs[i] }

func (h *nodeHeap) Push(x interface{}) {
	h.nodeIDs = append(h.nodeIDs, x.(int))
}

func (h *nodeHeap) Pop() interface{} {
	x := h.nodeIDs[len(h.nodeIDs)-1]
	h.nodeIDs = h.nodeIDs[0 : len(h.nodeIDs)-1]
	return x
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
