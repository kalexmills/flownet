package flownet_test

import (
	"strings"
	"testing"

	"github.com/kalexmills/flownet"
)

func TestSanityAllFlowNetworks(t *testing.T) {
	visitAllInstances(t, func(t *testing.T, path string, instance TestInstance) error {
		graph := flownet.NewFlowNetwork(instance.numNodes)
		for edge, cap := range instance.capacities {
			if err := graph.AddEdge(edge.from, edge.to, cap); err != nil {
				t.Error(err)
			}
		}
		graph.PushRelabel()
		outflow := graph.Outflow()
		t.Logf("test %s reported max flow of %d", path, outflow)
		if outflow == 0 {
			t.Errorf("failed test %s, expected non-zero max flow", path)
		}
		if instance.expectedFlow == -1 { // run sanity checks for any instance we don't know the max-flow value of
			if err := flownet.SanityChecks.FlowNetwork(graph, true); err != nil {
				t.Errorf("sanity checks failed: %v", err)
				return err
			}
			return nil
		}
		if instance.expectedFlow != outflow {
			t.Errorf("failed test %s expected max-flow of %d but was %d", path, instance.expectedFlow, outflow)
			return nil
		}
		return nil
	})
}

func TestTopSortAllFlowNetworks(t *testing.T) {
	visitAllInstances(t, func(t *testing.T, path string, instance TestInstance) error {
		graph := flownet.NewFlowNetwork(instance.numNodes)
		for edge, cap := range instance.capacities {
			if err := graph.AddEdge(edge.from, edge.to, cap); err != nil {
				t.Error(err)
			}
		}
		order, err := flownet.TopSort(graph, func(x, y int) bool { return x < y })
		if strings.Contains(path, "cycle") || strings.Contains(path, "graph1") {
			if err == nil {
				t.Errorf("failed %s: expected topological sort to report a cycle", path)
			}
			return nil
		}
		if err != nil {
			t.Errorf("failed %s: did not expect topological sort to report a cycle", path)
		}
		for edge := range instance.capacities {
			for _, nodeID := range order {
				if nodeID == edge.from {
					break
				}
				if nodeID == edge.to {
					t.Errorf("failed %s: found destination of edge %v before its source", path, edge)
				}
			}
		}
		return nil
	})
}
