//go:build goexperiment.regions

package region

import (
	. "experiments/benchmarks/metrics"
	"region"
	"runtime/debug"
	"time"
)

type value struct {
	x            [32]int
	latencyStart time.Time
}

func producing(buffer chan value, done chan bool, op int, valueRange int, r *region.Region) {
	r2 := region.CreateRegion(280 * op)
	for i := region.AllocFromRegion[int](r2); *i < op; *i++ {
		allocationStart := time.Now()
		x := region.AllocFromRegion[value](r2)
		AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

		x.latencyStart = time.Now()

		buffer <- *x
	}

	deallocationStart := time.Now()
	r2.RemoveRegion()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	r.DecRefCounter()
	done <- true
}

func consuming(buffer chan value, done chan bool, r *region.Region) {
	for x := range buffer {
		Latency.Add(time.Since(x.latencyStart).Nanoseconds())
		_ = x
	}

	r.DecRefCounter()
	done <- true
}

func RunProducerConsumer(valueRange int) SystemMetrics {
	debug.SetGCPercent(-1)

	ComputationTime.Store(0)
	AllocationTime.Store(0)
	DeallocationTime.Store(0)
	Latency.Store(0)

	computationTimeStart := time.Now()
	r1 := region.CreateRegion(0)

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

	deallocationStart := time.Now()
	r1.RemoveRegion()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	ComputationTime.Store(time.Since(computationTimeStart).Nanoseconds())

	return SystemMetrics{
		float64(ComputationTime.Load()),
		float64(ProConOp*Goroutines) / float64(ComputationTime.Load()),
		float64(Latency.Load()),
		float64(AllocationTime.Load()),
		float64(DeallocationTime.Load())}
}
