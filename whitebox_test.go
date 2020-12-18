package flownet

import (
	"math"
	"testing"
)

func TestSetNodeOrder(t *testing.T) {
	tests := []struct {
		networkSize int
		nodeList    []int
		expectError bool
	}{
		{5, []int{4, 3, 2, 1, 0}, false},
		{5, []int{4, 3, 2, 1}, true},
		{5, []int{4, 3, 2, 1, 5}, true},
		{5, []int{4, 3, 2, 1, 4}, true},
		{5, []int{5, 4, 3, 2, 1, 0}, true},
		{5, []int{4, 3, 2, 1, 0, -1}, true},
	}
	for idx, test := range tests {
		g := NewFlowNetwork(test.networkSize)
		err := g.SetNodeOrder(test.nodeList)
		if err == nil && test.expectError {
			t.Errorf("test #%d: expected error, but found none", idx)
		}
		if err != nil && !test.expectError {
			t.Errorf("test #%d: unexpected error %v", idx, err)
		}
		n := len(test.nodeList)
		for i, x := range g.nodeOrder {
			if x != test.nodeList[n-1-i]+2 {
				t.Errorf("test #%d: expected node order idx %d to be %d, but was %d", idx, i, test.nodeList[i], x)
			}
		}
	}
}

func TestAddEdge_ManualSourceSink(t *testing.T) {
	tests := []struct {
		edgesToAdd         [][]int
		expectManualSource bool
		expectManualSink   bool
	}{
		{[][]int{{Source, 0}, {0, Sink}}, true, true},
		{[][]int{{0, Sink}}, false, true},
		{[][]int{{Source, 0}}, true, false},
		{[][]int{{1, 0}}, false, false},
	}

	for idx, test := range tests {
		g := NewFlowNetwork(2)

		for _, edge := range test.edgesToAdd {
			err := g.AddEdge(edge[0], edge[1], 1)
			if err != nil {
				t.Errorf("test #%d: expected no error but found: %v", idx, err)
			}
		}
		if g.manualSink != test.expectManualSink {
			t.Errorf("test #%d: g.manualSink == %t, expected %t", idx, !test.expectManualSink, test.expectManualSink)
		}
		if g.manualSource != test.expectManualSource {
			t.Errorf("test #%d: g.manualSource == %t, expected %t", idx, !test.expectManualSource, test.expectManualSource)
		}
	}
}

func TestAddNode_ManualSourceSink(t *testing.T) {
	tests := []struct {
		manualSource           bool
		manualSink             bool
		expectedSourceCapacity int64
		expectedSinkCapacity   int64
	}{
		{false, false, math.MaxInt64, math.MaxInt64},
		{false, true, math.MaxInt64, 0},
		{true, false, 0, math.MaxInt64},
		{true, true, 0, 0},
	}

	for idx, test := range tests {
		g := NewFlowNetwork(2)
		g.manualSink = test.manualSink
		g.manualSource = test.manualSource
		u := g.AddNode()
		if g.Capacity(u, Sink) != test.expectedSinkCapacity {
			t.Errorf("test #%d: expected sink capacity %v", idx, test.expectedSinkCapacity)
		}
		if g.Capacity(Source, u) != test.expectedSourceCapacity {
			t.Errorf("test #%d: expected source capacity %v", idx, test.expectedSourceCapacity)
		}
	}
}
