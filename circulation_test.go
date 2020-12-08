package flownet_test

import (
	"testing"

	"github.com/kalexmills/flownet"
)

func TestSanityCheckAllCirculations(t *testing.T) {
	visitAllInstances(t, func(t *testing.T, path string, instance TestInstance) error {
		graph := flownet.NewCirculation(instance.numNodes)
		for edge, cap := range instance.capacities {
			if edge.from < 0 || edge.to < 0 {
				continue
			}
			if err := graph.AddEdge(edge.from, edge.to, cap, 1); err != nil {
				t.Error(err)
			}
		}
		graph.PushRelabel()
		outflow := graph.Outflow()
		t.Logf("test %s had a flow of %d", path, outflow)
		if err := flownet.SanityChecks.Circulation(graph); err != nil {
			t.Errorf("sanity checks failed: %v", err)
			return err
		}
		return nil
	})
}
