package flownet_test

import (
	"fmt"

	"github.com/kalexmills/flownet"
)

// Demonstrates how to use a flow network to compute max-flow.
func ExampleFlowNetwork() {
	fn := flownet.NewFlowNetwork(6) // allocates a flow network with nodeIDs 0, 1, ..., 5

	type edge struct {
		source, target int
		capacity       int64
	}

	edges := []edge{
		{0, 1, 15}, {0, 2, 4}, {1, 3, 12}, {3, 2, 3}, {2, 4, 10},
		{4, 1, 5}, {4, 5, 10}, {3, 5, 7},
	}

	for _, edge := range edges {
		// adds an edge between nodes with the provided capacity
		fn.AddEdge(edge.source, edge.target, edge.capacity)
	}

	fn.PushRelabel() // run max-flow

	fmt.Printf("found max flow of %d = 14\n", fn.Outflow())

	for _, e := range edges {
		flow := fn.Flow(e.source, e.target)
		capacity := fn.Capacity(e.source, e.target)
		fmt.Printf("\tedge %d -> %d:  flow = %d / %d\n", e.source, e.target, flow, capacity)
	}
	// Output:
	// found max flow of 14 = 14
	// 	edge 0 -> 1:  flow = 10 / 15
	// 	edge 0 -> 2:  flow = 4 / 4
	// 	edge 1 -> 3:  flow = 10 / 12
	// 	edge 3 -> 2:  flow = 3 / 3
	// 	edge 2 -> 4:  flow = 7 / 10
	// 	edge 4 -> 1:  flow = 0 / 5
	// 	edge 4 -> 5:  flow = 7 / 10
	// 	edge 3 -> 5:  flow = 7 / 7
}

// Demonstrates how to use a circulation to set lower-bounds on edges.
func ExampleCirculation() {
	c := flownet.NewCirculation(6)
	type edge struct {
		source, target   int
		capacity, demand int64
	}
	// a circulation allows for demand values on the edges of the flow network.
	edges := []edge{
		{0, 1, 15, 0}, {0, 2, 4, 0}, {1, 3, 12, 0}, {3, 2, 3, 0}, {2, 4, 10, 0},
		{4, 1, 5, 4}, {4, 5, 10, 0}, {3, 5, 7, 0},
	}
	for _, edge := range edges {
		c.AddEdge(edge.source, edge.target, edge.capacity, edge.demand)
	}

	c.SetNodeDemand(0, -4)
	c.SetNodeDemand(5, 4)

	c.PushRelabel()

	// there is no notion of source or sink in a circulation; the only question
	// is whether there is a flow which satisfies the requested demand.
	fmt.Printf("demand satisfied: %t\n", c.SatisfiesDemand())

	// the outflow for a circulation is the total amount of flow circulating
	// through the network.
	fmt.Printf("total flow: %d\n", c.Outflow())
	for _, e := range edges {
		flow := c.Flow(e.source, e.target)
		capacity := c.Capacity(e.source, e.target)
		demand := c.EdgeDemand(e.source, e.target)
		fmt.Printf("\tedge %d -> %d:  flow = %d / %d\tdemand = %d\n", e.source, e.target, flow, capacity, demand)
	}
	//Output:
	// demand satisfied: true
	// total flow: 4
	// 	edge 0 -> 1:  flow = 3 / 15	demand = 0
	// 	edge 0 -> 2:  flow = 1 / 4	demand = 0
	// 	edge 1 -> 3:  flow = 7 / 12	demand = 0
	// 	edge 3 -> 2:  flow = 3 / 3	demand = 0
	// 	edge 2 -> 4:  flow = 4 / 10	demand = 0
	// 	edge 4 -> 1:  flow = 4 / 5	demand = 4
	// 	edge 4 -> 5:  flow = 0 / 10	demand = 0
	// 	edge 3 -> 5:  flow = 4 / 7	demand = 0
}
