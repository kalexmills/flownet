package flownet

import "testing"

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
