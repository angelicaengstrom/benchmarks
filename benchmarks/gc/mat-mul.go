package gc

import (
	. "experiments/benchmarks/metrics"
	"math/rand/v2"
	"runtime"
	"runtime/debug"
	"time"
)

type product struct {
	res int
	pos position
}

type position struct {
	x            int
	y            int
	latencyStart time.Time
}

func generateMatrix(valueRange int) [][]int {
	allocationStart := time.Now()
	matrix := make([][]int, Rows)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for i := new(int); *i < len(matrix); *i++ {

		allocationStart = time.Now()
		matrix[*i] = make([]int, Cols)
		AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

		for j := new(int); *j < len(matrix[*i]); *j++ {
			matrix[*i][*j] = rand.IntN(valueRange)
		}
	}
	return matrix
}

func matrixMultiplication(m1 [][]int, m2 [][]int) [][]int {
	if len(m1[0]) != len(m2) {
		return nil
	}
	allocationStart := time.Now()
	r1 := new(int)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())
	*r1 = len(m1)

	allocationStart = time.Now()
	c2 := new(int)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())
	*c2 = len(m2[0])

	allocationStart = time.Now()
	result := make([][]int, *r1)
	products := make(chan product, (*r1)*(*c2))
	positions := make(chan position, (*r1)*(*c2))
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for i := new(int); *i < Goroutines; *i++ {
		go calculateProducts(m1, m2, products, positions)
	}

	for i := new(int); *i < len(result); *i++ {
		allocationStart = time.Now()
		result[*i] = make([]int, *c2)
		AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

		for j := new(int); *j < *c2; *j++ {
			positions <- position{*i, *j, time.Now()}
		}
	}

	for i := new(int); *i < (*r1)*(*c2); *i++ {
		r := <-products
		result[r.pos.x][r.pos.y] = r.res
	}

	close(positions)

	return result
}

func calculateProduct(row []int, col []int) *int {
	allocationStart := time.Now()
	p := new(int)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())
	*p = 0
	for k := new(int); *k < len(row); *k++ {
		*p += row[*k] * col[*k]
	}
	return p
}

func fetchColumn(m2 [][]int, col []int, j int) []int {
	for i := new(int); *i < len(m2); *i++ {
		col[*i] = m2[*i][j]
	}
	return col
}

func calculateProducts(m1 [][]int, m2 [][]int, products chan product, positions chan position) {
	allocationStart := time.Now()
	var col = make([]int, len(m2))
	var pos = new(position)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for *pos = range positions {
		Latency.Add(time.Since(pos.latencyStart).Nanoseconds())
		col = fetchColumn(m2, col, pos.y)
		products <- product{
			res: *calculateProduct(m1[pos.x], col),
			pos: *pos,
		}

	}
}

func RunMatrixMultiplication(valueRange int) Metrics {
	debug.SetGCPercent(-1)

	AllocationTime.Store(0)
	DeallocationTime.Store(0)
	Latency.Store(0)

	start := time.Now()

	m1 := generateMatrix(valueRange)
	m2 := generateMatrix(valueRange)

	matrixMultiplication(m1, m2)

	computationTime := float64(time.Since(start).Nanoseconds())

	// end timers here
	deallocationStart := time.Now()
	runtime.GC()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	throughput := float64(Rows*Cols*1_000_000) / float64(computationTime)

	// TODO: HeapIntfrag Doesn't work, check compiler
	return Metrics{float64(computationTime) / 1_000,
		throughput,
		float64(Latency.Load()) / 1_000,
		float64(AllocationTime.Load()) / 1_000,
		float64(DeallocationTime.Load()) / 1_000}
}
