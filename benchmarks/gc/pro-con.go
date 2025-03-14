package gc

import (
	. "experiments/benchmarks/metrics"
	"math/rand/v2"
	"runtime"
	"runtime/debug"
	"time"
)

type value struct {
	x            int
	latencyStart time.Time
}

func producing(buffer chan value, done chan bool, op int, valueRange int) {
	for i := new(int); *i < op; *i++ {
		buffer <- value{rand.IntN(valueRange), time.Now()}
	}
	done <- true
}

func consuming(buffer chan value, done chan bool) {
	allocationStart := time.Now()
	var x = new(value)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for *x = range buffer {
		Latency.Add(time.Since(x.latencyStart).Nanoseconds())
	}
	done <- true
}

func RunProducerConsumer(valueRange int) Metrics {
	debug.SetGCPercent(-1)

	ComputationTime.Store(0)
	AllocationTime.Store(0)
	DeallocationTime.Store(0)
	Latency.Store(0)

	allocationStart := time.Now()
	buffer := make(chan value)
	doneProducers := make(chan bool)
	doneConsumers := make(chan bool)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	computationTimeStart := time.Now()

	for i := 0; i < Goroutines; i++ {
		go producing(buffer, doneProducers, ProConOp, valueRange)
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

	ComputationTime.Store(time.Since(computationTimeStart).Nanoseconds())

	deallocationStart := time.Now()
	runtime.GC()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	return Metrics{
		float64(ComputationTime.Load()) / 1_000,
		float64(ProConOp*1_000_000) / float64(ComputationTime.Load()),
		float64(Latency.Load()) / 1_000,
		float64(AllocationTime.Load()) / 1_000,
		float64(DeallocationTime.Load()) / 1_000}
}
