package flownet_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/kalexmills/flownet"
)

func TestSanityAllCirculations(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	visitAllInstances(t, CircInstances, func(t *testing.T, path string, instance TestInstance) error {
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
			demand, ok := instance.demands[edge]
			if !ok {
				demand = 0
			}
			if err := graph.AddEdge(edge.from, edge.to, cap, demand); err != nil {
				t.Error(err)
			}
		}
		graph.PushRelabel()
		outflow := graph.Outflow()
		if instance.expectedFlow != -1 {
			if instance.expectedFlow > outflow {
				t.Errorf("expected at least %d units of flow, found %d", instance.expectedFlow, outflow)
			}
		}
		t.Logf("test %s had a flow of %d; satisfied demand? %t", path, outflow, graph.SatisfiesDemand())
		if err := flownet.SanityChecks.Circulation(graph); err != nil {
			t.Errorf("sanity checks failed: %v", err)
			return err
		}
		return nil
	})
}
