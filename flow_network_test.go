package flownet_test

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/kalexmills/flownet"
)

func TestAllTestData(t *testing.T) {
	filepath.Walk("testdata", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".dat") {
			return nil
		}
		f, err := os.Open(path)
		defer f.Close()
		if err != nil {
			return err
		}
		instance, err := loadInstance(f)
		if err != nil {
			return err
		}
		return runTest(t, filepath.Base(path), instance)
	})
}

func runTest(t *testing.T, path string, instance TestInstance) error {
	graph := flownet.NewFlowNetwork(instance.numNodes)
	for edge, cap := range instance.capacities {
		if err := graph.AddEdge(edge.from, edge.to, cap); err != nil {
			t.Error(err)
		}
	}
	graph.PushRelabel()
	outflow := graph.Outflow()
	if instance.expectedFlow == -1 { // run sanity checks
		if err := flownet.SanityChecks(graph); err != nil {
			t.Errorf("sanity checks failed: %v", err)
			return err
		}
		return nil
	}
	if instance.expectedFlow != outflow {
		t.Errorf("failed test %s expected max-flow of %d but was %d", path, instance.expectedFlow, outflow)
	}
	return nil
}

// loadInstance loads a test instance. Each test is a UTF-8 encoded file. Each line of the file consists of
// integers separated by a single space character. The first line of the file contains a single integer describing
// the expected max flow which is attainable for the test instance. All remaining lines of the file are either empty
// or consist of 3 integers describing one directed edge of the flow network. The first two integers are the source
// and destination nodes of the edge, respectively, while the third integer is the maximum capacity of the edge.
func loadInstance(reader io.Reader) (TestInstance, error) {
	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() {
		return TestInstance{}, scanner.Err()
	}
	expectedFlow, err := strconv.ParseInt(scanner.Text(), 10, 32)
	if err != nil {
		return TestInstance{}, fmt.Errorf("first line of file must consist of a single integer: %w", err)
	}
	result := TestInstance{
		expectedFlow: expectedFlow,
		capacities:   make(map[Edge]int64),
	}
	maxNodeId := 0
	for scanner.Scan() {
		if scanner.Text() == "" {
			continue
		}
		fields := strings.Split(scanner.Text(), " ")
		if len(fields) != 3 {
			return TestInstance{}, fmt.Errorf("expected 3 space-separated fields on line reading: %s", scanner.Text())
		}
		ints, err := parseInts(fields)
		if err != nil {
			return TestInstance{}, fmt.Errorf("could not parse line as integers: %w", err)
		}
		result.capacities[Edge{ints[0], ints[1]}] = int64(ints[2])
		maxNodeId = max(max(maxNodeId, ints[0]), ints[1])
	}
	result.numNodes = maxNodeId + 1
	return result, nil
}

func parseInts(strs []string) ([]int, error) {
	result := make([]int, 0, len(strs))
	for _, str := range strs {
		i, err := strconv.ParseInt(str, 10, 32)
		if err != nil {
			return nil, err
		}
		result = append(result, int(i))
	}
	return result, nil
}

type TestInstance struct {
	numNodes     int
	expectedFlow int64
	capacities   map[Edge]int64
}

type Edge struct {
	from, to int
}

func max(x, y int) int {
	if x < y {
		return y
	}
	return x
}
