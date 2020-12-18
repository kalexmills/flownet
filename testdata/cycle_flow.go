// cycle flow generates random tests by adding cycles to a graph.
package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/kalexmills/flownet"
)

func main() {
	rand.Seed(time.Now().Unix())

	for idx, sizes := range [][]int{uniformCycles(20, 10), uniformCycles(10, 20)} {
		capacities, expected := makeCyclic(100, sizes...)
		writeFile(fmt.Sprintf("cycles_small_%d.circ", idx), capacities, expected)
	}
	for idx, sizes := range [][]int{uniformCycles(50, 50), uniformCycles(60, 40), uniformCycles(70, 30)} {
		capacities, expected := makeCyclic(100, sizes...)
		writeFile(fmt.Sprintf("cycles_medium_%d.circ", idx), capacities, expected)
	}
}

func uniformCycles(n, maxLength int) []int {
	result := make([]int, n)
	for i := 0; i < n; i++ {
		result[i] = rand.Intn(maxLength-2) + 2
	}
	return result
}

// makeCyclic creates a graph formed by layering a bunch of cycles ontop of one another. Each
// size entry is the length of a directed cycle to add to the graph. Each cycle receives a random
// weight from 1 to 10. 10% of the nodes are connected to the source and sink with capacity 1.
func makeCyclic(nodes int, sizes ...int) (capacities, int64) {
	result := make(capacities)
	// add cycles
	for _, length := range sizes {
		if length > nodes {
			length = nodes
		}
		perm := rand.Perm(nodes)
		weight := 1 + rand.Int63n(10)
		for i := 0; i < length-1; i++ {
			delete(result, edge{perm[i+1], perm[i]})
			result[edge{perm[i], perm[i+1]}] = weight
		}
		delete(result, edge{perm[0], perm[length-1]})
		result[edge{perm[length-1], perm[0]}] = weight
	}
	// connect nodes to source/sink
	perm := rand.Perm(nodes) // form a random cycle by randomly permuting the nodes
	for i := 0; i < nodes/10; i++ {
		if rand.Float32() < 0.5 {
			result[edge{flownet.Source, perm[i]}] = 10
		} else {
			result[edge{perm[i], flownet.Sink}] = 10
		}
	}
	return result, -1
}

type capacities map[edge]int64

type edge struct {
	from, to int
}

func min(x, y int64) int64 {
	if x < y {
		return x
	}
	return y
}

func writeFile(name string, capacities capacities, expected int64) {
	f, err := os.OpenFile(name, os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0666)
	defer f.Close()
	if err != nil {
		log.Fatalln(err)
	}
	writeOutput(f, capacities, expected)
}

func writeOutput(out io.Writer, capacities capacities, expectedFlow int64) {
	fmt.Fprintf(out, "%d\n", expectedFlow)
	for edge, cap := range capacities {
		fmt.Fprintf(out, "%d %d %d\n", edge.from, edge.to, cap)
	}
}
