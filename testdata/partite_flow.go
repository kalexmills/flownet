// partite_flow generates random multipartite flow networks for testing.
package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"time"
)

func main() {
	rand.Seed(time.Now().Unix())
	for idx, sizes := range [][]int{{10, 10}, {100, 10}, {10, 100}} {
		capacities, expected := makeMultipartite(sizes...)
		if len(sizes) > 2 {
			expected = -1
		}
		writeFile(fmt.Sprintf("bipartite_%d.flow", idx), capacities, expected)
	}
	for idx, sizes := range [][]int{{10, 100, 10}, {100, 10, 100}, {10, 10, 10}} {
		capacities, expected := makeMultipartite(sizes...)
		if len(sizes) > 2 {
			expected = -1
		}
		writeFile(fmt.Sprintf("tripartite_%d.flow", idx), capacities, expected)
	}
	for idx, sizes := range [][]int{{50, 100, 50, 100, 50, 100}} {
		capacities, expected := makeMultipartite(sizes...)
		if len(sizes) > 2 {
			expected = -1
		}
		writeFile(fmt.Sprintf("multipartite_medium_%d.flow", idx), capacities, expected)
	}
}

func writeFile(name string, capacities capacities, expected int64) {
	f, err := os.OpenFile(name, os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0666)
	defer f.Close()
	if err != nil {
		log.Fatalln(err)
	}
	writeOutput(f, capacities, expected)
}

type capacities map[edge]int64

// makeMultipartite creates a multipartite graph, which is a few bipartite graphs connected end-to-end.
// The flow network should find a max flow equal to the minimum number of edges crossing any partition.
func makeMultipartite(sizes ...int) (capacities, int64) {
	if len(sizes) <= 1 {
		return make(capacities), 0
	}
	capacities := make(capacities)
	minEdgeCount := int64(math.MaxInt64)
	minSIdx := 0
	for k := 1; k < len(sizes); k++ {
		minTIdx := minSIdx + sizes[k-1]
		maxTIdx := minTIdx + sizes[k]
		edgeCount := int64(0)
		for i := minSIdx; i < minTIdx; i++ {
			for j := minTIdx; j < maxTIdx; j++ {
				if rand.Float32() < 0.5 {
					capacities[edge{i, j}] = 1
					edgeCount++
				}
			}
		}
		minSIdx += sizes[k-1]
		minEdgeCount = min(minEdgeCount, edgeCount)
	}
	return capacities, minEdgeCount
}

type edge struct {
	from, to int
}

func min(x, y int64) int64 {
	if x < y {
		return x
	}
	return y
}

func writeOutput(out io.Writer, capacities capacities, expectedFlow int64) {
	fmt.Fprintf(out, "%d\n", expectedFlow)
	for edge, cap := range capacities {
		fmt.Fprintf(out, "%d %d %d\n", edge.from, edge.to, cap)
	}
}
