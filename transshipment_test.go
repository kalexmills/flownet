package flownet_test

import (
	"testing"

	"github.com/kalexmills/flownet"
)

func TestSanityCheckAllTransshipments(t *testing.T) {
	visitAllInstances(t, func(t *testing.T, path string, instance TestInstance) error {
		graph := flownet.NewTransshipment(instance.numNodes)
		for edge, cap := range instance.capacities {
			if edge.from < 0 || edge.to < 0 {
				continue
			}
			if err := graph.AddEdge(edge.from, edge.to, cap, 1); err != nil {
				t.Error(err)
			}
		}
		for i := 0; i < instance.numNodes; i++ {
			graph.SetNodeBounds(i, 0, 3)
		}
		graph.PushRelabel()
		outflow := graph.Outflow()
		t.Logf("test %s had a flow of %d", path, outflow)
		if err := flownet.SanityChecks.Transshipment(graph); err != nil {
			t.Errorf("sanity checks failed in %s: %v", path, err)
			return err
		}
		return nil
	})
}
