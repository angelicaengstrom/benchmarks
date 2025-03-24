//go:build goexperiment.regions

package region

import (
	. "experiments/benchmarks/metrics"
	"math/rand/v2"
	"region"
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

func generateMatrix(valueRange int, r *region.Region) [Rows][Cols]int {
	allocationStart := time.Now()
	matrix := region.AllocFromRegion[[Rows][Cols]int](r)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for i := region.AllocFromRegion[int](r); *i < Rows; *i++ {
		for j := region.AllocFromRegion[int](r); *j < Cols; *j++ {
			(*matrix)[*i][*j] = rand.IntN(valueRange) + 1
		}
	}

	return *matrix
}

func matrixMultiplication(m1 [Rows][Cols]int, m2 [Rows][Cols]int, r1 *region.Region) [Rows][Cols]int {
	if Cols != Rows {
		return [Rows][Cols]int{}
	}

	allocationStart := time.Now()
	result := region.AllocFromRegion[[Rows][Cols]int](r1)
	products := region.AllocChannel[product](Rows*Cols, r1)
	positions := region.AllocChannel[position](Rows*Cols, r1)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for i := region.AllocFromRegion[int](r1); *i < Goroutines; *i++ {
		if r1.IncRefCounter() {
			go calculateProducts(m1, m2, products, positions, r1)
		}
	}

	for i := region.AllocFromRegion[int](r1); *i < Rows; *i++ {
		for j := region.AllocFromRegion[int](r1); *j < Cols; *j++ {
			positions <- position{*i, *j, time.Now()}
		}
	}

	for i := region.AllocFromRegion[int](r1); *i < (Rows)*(Cols); *i++ {
		r := <-products
		result[r.pos.x][r.pos.y] = r.res
	}

	close(positions)

	return *result
}

func calculateProduct(row [Cols]int, col [Rows]int, r *region.Region) *int {
	allocationStart := time.Now()
	p := region.AllocFromRegion[int](r)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	*p = 0
	for k := region.AllocFromRegion[int](r); *k < len(row); *k++ {
		*p += row[*k] * col[*k]
	}
	return p
}

func fetchColumn(m2 [Rows][Cols]int, j int, col [Rows]int, r *region.Region) [Rows]int {
	for i := region.AllocFromRegion[int](r); *i < len(m2); *i++ {
		col[*i] = m2[*i][j]
	}
	return col
}

func calculateProducts(
	m1 [Rows][Cols]int,
	m2 [Rows][Cols]int,
	products chan product,
	positions chan position,
	r1 *region.Region) {

	allocationStart := time.Now()
	col := region.AllocFromRegion[[Rows]int](r1)
	pos := region.AllocFromRegion[position](r1)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for *pos = range positions {
		Latency.Add(time.Since(pos.latencyStart).Nanoseconds())

		fetchColumn(m2, pos.y, *col, r1)

		products <- product{
			res: *calculateProduct(m1[pos.x], *col, r1),
			pos: *pos,
		}
	}

	r1.DecRefCounter()
}

func RunMatrixMultiplication(valueRange int) Metrics {
	debug.SetGCPercent(-1)
	AllocationTime.Store(0)
	DeallocationTime.Store(0)
	Latency.Store(0)

	start := time.Now()
	r1 := region.CreateRegion()

	m1 := generateMatrix(valueRange, r1)
	m2 := generateMatrix(valueRange, r1)

	matrixMultiplication(m1, m2, r1)

	computationTime := float64(time.Since(start).Nanoseconds())

	deallocationStart := time.Now()
	r1.RemoveRegion()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	throughput := float64(Rows*Cols*1_000_000) / float64(computationTime)

	return Metrics{
		float64(computationTime) / 1_000,
		throughput,
		float64(Latency.Load()) / 1_000,
		float64(AllocationTime.Load()) / 1_000,
		float64(DeallocationTime.Load()) / 1_000}
}
