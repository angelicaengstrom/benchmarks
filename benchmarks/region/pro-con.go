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

type value struct {
	x            int
	latencyStart time.Time
}

func producing(buffer chan value, done chan bool, op int, valueRange int, r *region.Region) {
	for i := region.AllocFromRegion[int](r); *i < op; *i++ {
		buffer <- value{rand.IntN(valueRange), time.Now()}
	}
	done <- true
	r.DecRefCounter()
}

func consuming(buffer chan value, done chan bool, r *region.Region) {
	allocationStart := time.Now()
	x := region.AllocFromRegion[value](r)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for *x = range buffer {
		Latency.Add(time.Since(x.latencyStart).Nanoseconds())
		_ = x
	}
	done <- true
	r.DecRefCounter()
}

func RunProducerConsumer(valueRange int) Metrics {
	debug.SetGCPercent(-1)

	ComputationTime.Store(0)
	AllocationTime.Store(0)
	DeallocationTime.Store(0)
	Latency.Store(0)

	computationTimeStart := time.Now()
	r1 := region.CreateRegion()

	allocationStart := time.Now()
	buffer := region.AllocChannel[value](0, r1)
	doneProducers := region.AllocChannel[bool](0, r1)
	doneConsumers := region.AllocChannel[bool](0, r1)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for i := 0; i < Goroutines; i++ {
		if r1.IncRefCounter() {
			go producing(buffer, doneProducers, ProConOp, valueRange, r1)
		}
		if r1.IncRefCounter() {
			go consuming(buffer, doneConsumers, r1)
		}
	}

	if r1.IncRefCounter() {
		go func() {
			for i := 0; i < Goroutines; i++ {
				<-doneProducers
			}
			close(buffer)
			r1.DecRefCounter()
		}()
	}

	for i := 0; i < Goroutines; i++ {
		<-doneConsumers
	}

	ComputationTime.Store(time.Since(computationTimeStart).Nanoseconds())

	deallocationStart := time.Now()
	r1.RemoveRegion()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	runtime.GC()

	return Metrics{
		float64(ComputationTime.Load()) / 1_000,
		float64(ProConOp*1_000_000) / float64(ComputationTime.Load()),
		float64(Latency.Load()) / 1_000,
		float64(AllocationTime.Load()) / 1_000,
		float64(DeallocationTime.Load()) / 1_000}
}
