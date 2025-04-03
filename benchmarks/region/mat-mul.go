//go:build goexperiment.regions

package region

import (
	. "experiments/benchmarks/metrics"
	"math/rand/v2"
	"region"
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

func generateMatrix(valueRange int, r *region.Region) [Rows]*[Cols]int {
	allocationStart := time.Now()
	matrix := region.AllocFromRegion[[Rows]*[Cols]int](r)
	i := region.AllocFromRegion[int](r)
	j := region.AllocFromRegion[int](r)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for *i = 0; *i < Rows; *i++ {
		allocationStart = time.Now()
		(*matrix)[*i] = region.AllocFromRegion[[Cols]int](r)
		AllocationTime.Add(time.Since(allocationStart).Nanoseconds())
		for *j = 0; *j < Cols; *j++ {
			(*matrix)[*i][*j] = rand.IntN(valueRange) + 1
		}
	}

	return *matrix
}

func matrixMultiplication(m1 [Rows]*[Cols]int, m2 [Rows]*[Cols]int, done chan bool, r1 *region.Region) {
	if Cols != Rows {
		return
	}

	r2 := region.CreateRegion((Rows + 1) * Cols * 8)

	allocationStart := time.Now()
	result := region.AllocFromRegion[[Rows]*[Cols]int](r2)
	products := region.AllocChannel[product](0, r1)
	positions := region.AllocChannel[position](0, r1)
	i := region.AllocFromRegion[int](r1)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for *i = 0; *i < Goroutines; *i++ {
		if r1.IncRefCounter() {
			go calculateProducts(m1, m2, products, positions, done, r1)
		}
	}

	for *i = 0; *i < Rows; *i++ {
		allocationStart = time.Now()
		(*result)[*i] = region.AllocFromRegion[[Cols]int](r2)
		AllocationTime.Add(time.Since(allocationStart).Nanoseconds())
	}

	r1.IncRefCounter()
	go func() {
		allocationStart = time.Now()
		i := region.AllocFromRegion[int](r1)
		j := region.AllocFromRegion[int](r1)
		AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

		for *i = 0; *i < Rows; *i++ {
			for *j = 0; *j < Cols; *j++ {
				positions <- position{*i, *j, time.Now()}
			}
		}
		r1.DecRefCounter()
		done <- true
	}()

	for *i = 0; *i < (Rows)*(Cols); *i++ {
		r := <-products
		(*result)[r.pos.x][r.pos.y] = r.res
	}

	close(positions)
	<-done

	deallocationStart := time.Now()
	r2.RemoveRegion()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())
}

func calculateProduct(row [Cols]int, col [Rows]int, k *int, p *int) *int {
	*p = 0
	for *k = 0; *k < len(row); *k++ {
		*p += row[*k] * col[*k]
	}
	return p
}

func fetchColumn(m2 [Rows]*[Cols]int, col [Rows]int, j int, i *int) [Rows]int {
	for *i = 0; *i < len(m2); *i++ {
		col[*i] = m2[*i][j]
	}
	return col
}

func calculateProducts(
	m1 [Rows]*[Cols]int,
	m2 [Rows]*[Cols]int,
	products chan product,
	positions chan position,
	done chan bool,
	r1 *region.Region) {

	r2 := region.CreateRegion(Rows)

	allocationStart := time.Now()
	col := region.AllocFromRegion[[Rows]int](r2)
	pos := region.AllocFromRegion[position](r2)
	i := region.AllocFromRegion[int](r2)
	p := region.AllocFromRegion[int](r2)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for *pos = range positions {
		Latency.Add(time.Since(pos.latencyStart).Nanoseconds())

		fetchColumn(m2, *col, pos.y, i)

		products <- product{
			res: *calculateProduct(*m1[pos.x], *col, i, p),
			pos: *pos,
		}
	}

	deallocationStart := time.Now()
	r2.RemoveRegion()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	r1.DecRefCounter()
	done <- true
}

func RunMatrixMultiplication(valueRange int) SystemMetrics {
	debug.SetGCPercent(-1)
	AllocationTime.Store(0)
	DeallocationTime.Store(0)
	Latency.Store(0)

	start := time.Now()

	r1 := region.CreateRegion((1 + Rows) * Cols * 16)
	done := region.AllocChannel[bool](0, r1)

	m1 := generateMatrix(valueRange, r1)
	m2 := generateMatrix(valueRange, r1)
	matrixMultiplication(m1, m2, done, r1)

	for i := 0; i < Goroutines; i++ {
		<-done
	}

	deallocationStart := time.Now()
	r1.RemoveRegion()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	computationTime := float64(time.Since(start).Nanoseconds())

	runtime.GC()

	throughput := float64(Rows*Cols) / float64(computationTime)

	return SystemMetrics{
		float64(computationTime),
		throughput,
		float64(Latency.Load()),
		float64(AllocationTime.Load()),
		float64(DeallocationTime.Load())}
}
