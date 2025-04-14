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

func generateMatrix(valueRange int) []*[]*int {
	allocationStart := time.Now()
	matrix := make([]*[]*int, Rows)
	j := new(int)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for i := new(int); *i < len(matrix); *i++ {
		allocationStart = time.Now()
		matrix[*i] = new([]*int)
		*matrix[*i] = make([]*int, Cols)
		AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

		for *j = 0; *j < len(*matrix[*i]); *j++ {
			AllocationTime.Add(time.Since(allocationStart).Nanoseconds())
			(*matrix[*i])[*j] = new(int)
			AllocationTime.Add(time.Since(allocationStart).Nanoseconds())
			*(*matrix[*i])[*j] = rand.IntN(valueRange)
		}
	}
	return matrix
}

func matrixMultiplication(m1 []*[]*int, m2 []*[]*int, done chan bool, result *[]*[]int) {
	if len(*m1[0]) != len(m2) {
		return
	}
	r1 := len(m1)
	c2 := len(*m2[0])

	allocationStart := time.Now()
	*result = make([]*[]int, r1)
	products := make(chan product)
	positions := make(chan position)
	i := new(int)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for *i = 0; *i < Goroutines; *i++ {
		go calculateProducts(m1, m2, products, positions, done)
	}

	allocationStart = time.Now()
	for *i = 0; *i < len(*result); *i++ {
		(*result)[*i] = new([]int)
		*(*result)[*i] = make([]int, c2)
	}
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	go func() {
		allocationStart = time.Now()
		i := new(int)
		j := new(int)
		AllocationTime.Add(time.Since(allocationStart).Nanoseconds())
		for *i = 0; *i < len(*result); *i++ {
			for *j = 0; *j < c2; *j++ {
				positions <- position{*i, *j, time.Now()}
			}
		}
		done <- true
	}()

	for *i = 0; *i < r1*c2; *i++ {
		r := <-products
		(*(*result)[r.pos.x])[r.pos.y] = r.res
	}

	close(positions)
	<-done
}

func calculateProduct(row []*int, col []*int, k *int, p *int) *int {
	*p = 0
	for *k = 0; *k < len(row); *k++ {
		*p += (*row[*k]) * (*col[*k])
	}
	return p
}

func fetchColumn(m2 []*[]*int, col []*int, j int, i *int) []*int {
	for *i = 0; *i < len(m2); *i++ {
		*col[*i] = *(*m2[*i])[j]
	}
	return col
}

func initColumn(col *[]*int, n int, i *int) {
	allocationStart := time.Now()
	for *i = 0; *i < n; *i++ {
		(*col)[*i] = new(int)
	}
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())
}

func calculateProducts(m1 []*[]*int, m2 []*[]*int, products chan product, positions chan position, done chan bool) {
	allocationStart := time.Now()
	col := make([]*int, len(m2))
	pos := new(position)
	i := new(int)
	p := new(int)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	initColumn(&col, len(m2), i)

	for *pos = range positions {
		Latency.Add(time.Since(pos.latencyStart).Nanoseconds())

		col = fetchColumn(m2, col, pos.y, i)

		products <- product{
			res: *calculateProduct(*m1[pos.x], col, i, p),
			pos: *pos,
		}
	}
	done <- true
}

func RunMatrixMultiplication(valueRange int) SystemMetrics {
	debug.SetGCPercent(-1)

	AllocationTime.Store(0)
	DeallocationTime.Store(0)
	Latency.Store(0)

	start := time.Now()

	allocationStart := time.Now()
	done := make(chan bool)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	m1 := generateMatrix(valueRange)
	m2 := generateMatrix(valueRange)

	var res []*[]int
	matrixMultiplication(m1, m2, done, &res)

	for i := 0; i < Goroutines; i++ {
		<-done
	}

	// end timers here
	deallocationStart := time.Now()
	runtime.GC()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	computationTime := float64(time.Since(start).Nanoseconds())

	throughput := float64(Rows*Cols) / float64(computationTime)

	return SystemMetrics{
		float64(computationTime),
		throughput,
		float64(Latency.Load()),
		float64(AllocationTime.Load()),
		float64(DeallocationTime.Load())}
}
