package gc

import (
	. "experiments/benchmarks/metrics"
	"runtime"
	"runtime/debug"
	"time"
)

type large struct {
	x            [32]int
	latencyStart time.Time
}

func producing(buffer chan large, done chan bool, x *large, i *int) {
	for i = new(int); *i < ProConOp; *i++ {
		allocationStart := time.Now()
		x = new(large)
		AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

		x.latencyStart = time.Now()

		buffer <- *x
	}
	done <- true
}

func consuming(buffer chan large, done chan bool) {
	for x := range buffer {
		Latency.Add(time.Since(x.latencyStart).Nanoseconds())
		_ = x
	}
	done <- true
}

func RunProducerConsumer(valueRange int) SystemMetrics {
	debug.SetGCPercent(-1)

	ComputationTime.Store(0)
	AllocationTime.Store(0)
	DeallocationTime.Store(0)
	Latency.Store(0)

	x := [Goroutines]large{} // To avoid escape analysis to the stack
	c := [Goroutines]int{}

	computationTimeStart := time.Now()

	allocationStart := time.Now()
	buffer := make(chan large)
	doneProducers := make(chan bool)
	doneConsumers := make(chan bool)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for i := 0; i < Goroutines; i++ {
		go producing(buffer, doneProducers, &x[i], &c[i])
		go consuming(buffer, doneConsumers)
	}

	go func() {
		for i := 0; i < Goroutines; i++ {
			<-doneProducers
		}
		close(buffer)
	}()

	for i := 0; i < Goroutines; i++ {
		<-doneConsumers
	}

	deallocationStart := time.Now()
	runtime.GC()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	ComputationTime.Store(time.Since(computationTimeStart).Nanoseconds())

	return SystemMetrics{
		float64(ComputationTime.Load()),
		float64(ProConOp*Goroutines) / float64(ComputationTime.Load()),
		float64(Latency.Load()),
		float64(AllocationTime.Load()),
		float64(DeallocationTime.Load())}
}
