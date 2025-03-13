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
	x int
	y int
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
			positions <- position{*i, *j}
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

func fetchColumn(m2 [][]int, j int) []int {
	allocationStart := time.Now()
	var col = make([]int, len(m2))
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for i := new(int); *i < len(m2); *i++ {
		col[*i] = m2[*i][j]
	}
	return col
}

func calculateProducts(m1 [][]int, m2 [][]int, products chan product, positions chan position) {
	var memStats runtime.MemStats
	latencyStart := time.Now()

	allocationStart := time.Now()
	var col = make([]int, len(m2))
	var pos = new(position)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for *pos = range positions {
		Latency.Add(time.Since(latencyStart).Nanoseconds())
		col = fetchColumn(m2, pos.y)
		products <- product{
			res: *calculateProduct(m1[pos.x], col),
			pos: *pos,
		}

		runtime.ReadMemStats(&memStats)

		if memStats.HeapAlloc > P_memoryConsuption.Load() {
			P_memoryConsuption.Store(memStats.HeapAlloc)
		}

		externalFrag := float64(memStats.HeapIdle) / float64(memStats.HeapSys)
		if externalFrag > P_externalFrag.Load().(float64) {
			P_externalFrag.Store(externalFrag)
		}

		internalFrag := float64(memStats.HeapIntFrag) / float64(memStats.HeapAlloc)
		if internalFrag > P_internalFrag.Load().(float64) {
			P_internalFrag.Store(internalFrag)
		}
		latencyStart = time.Now()
	}
}

func RunMatrixMultiplication(valueRange int) Metrics {
	AllocationTime.Store(0)
	DeallocationTime.Store(0)
	Latency.Store(0)
	P_memoryConsuption.Store(0)
	P_internalFrag.Store(0.0)
	P_externalFrag.Store(0.0)

	debug.SetGCPercent(-1)

	start := time.Now()

	m1 := generateMatrix(valueRange)
	m2 := generateMatrix(valueRange)

	matrixMultiplication(m1, m2)

	computationTime := float64(time.Since(start).Nanoseconds()) / float64(1_000_000_000)

	// end timers here
	deallocationStart := time.Now()
	runtime.GC()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	throughput := float64(Rows*Cols) / float64(computationTime)

	// TODO: HeapIntfrag Doesn't work, check compiler
	return Metrics{float64(computationTime),
		throughput,
		float64(Latency.Load()) / 1_000_000_000,
		float64(P_memoryConsuption.Load()),
		P_externalFrag.Load().(float64),
		P_internalFrag.Load().(float64),
		float64(AllocationTime.Load()) / 1_000_000_000,
		float64(DeallocationTime.Load()) / 1_000_000_000}
}
