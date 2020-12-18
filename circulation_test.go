package flownet_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/kalexmills/flownet"
)

func TestSanityAllCirculations(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	visitAllInstances(t, func(t *testing.T, path string, instance TestInstance) error {
		graph := flownet.NewCirculation(instance.numNodes)

		for edge, cap := range instance.capacities {
			if edge.from == flownet.Source {
				graph.SetNodeDemand(edge.to, -10)
			}
			if edge.to == flownet.Sink {
				graph.SetNodeDemand(edge.from, 10)
			}
			if edge.from < 0 || edge.to < 0 {
				continue
			}
			if cap <= 0 {
				continue
			}
			randomDemand := rand.Int63n(2)
			if err := graph.AddEdge(edge.from, edge.to, cap, randomDemand); err != nil {
				t.Error(err)
			}
		}
		graph.PushRelabel()
		outflow := graph.Outflow()
		t.Logf("test %s had a flow of %d", path, outflow)
		t.Logf("test %s satisfied demand? %t", path, graph.SatisfiesDemand())
		if err := flownet.SanityChecks.Circulation(graph); err != nil {
			t.Errorf("sanity checks failed: %v", err)
			return err
		}
		return nil
	})
}
