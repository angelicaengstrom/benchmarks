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

func producing(buffer chan int, done chan bool, op int, valueRange int, r *region.Region) {
	for i := region.AllocFromRegion[int](r); *i < op; *i++ {
		buffer <- rand.IntN(valueRange)
	}
	done <- true
	r.DecRefCounter()
}

func consuming(buffer chan int, done chan bool, r *region.Region) {
	allocationStart := time.Now()
	x := region.AllocFromRegion[int](r)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	var memStats runtime.MemStats
	latencyStart := time.Now()
	for *x = range buffer {
		Latency.Add(time.Since(latencyStart).Nanoseconds())
		_ = x
		latencyStart = time.Now()

		runtime.ReadMemStats(&memStats)

		if memStats.HeapAlloc > P_memoryConsuption.Load() {
			P_memoryConsuption.Store(memStats.HeapAlloc)
		}

		externalFrag := float64(memStats.HeapIdle) / float64(memStats.HeapSys)
		if externalFrag > P_externalFrag.Load().(float64) {
			P_externalFrag.Store(externalFrag)
		}

		internalFrag := float64(memStats.RegionIntFrag) / float64(memStats.RegionInUse)
		if internalFrag > P_internalFrag.Load().(float64) {
			P_internalFrag.Store(internalFrag)
		}
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
	P_memoryConsuption.Store(0)
	P_internalFrag.Store(0.0)
	P_externalFrag.Store(0.0)

	r1 := region.CreateRegion()

	allocationStart := time.Now()
	buffer := region.AllocChannel[int](0, r1)
	doneProducers := region.AllocChannel[bool](0, r1)
	doneConsumers := region.AllocChannel[bool](0, r1)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	computationTimeStart := time.Now()

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

	return Metrics{
		float64(ComputationTime.Load()) / 1_000_000_000,
		float64(ProConOp*1_000_000_000) / float64(ComputationTime.Load()),
		float64(Latency.Load()) / 1_000_000_000,
		float64(P_memoryConsuption.Load()),
		P_externalFrag.Load().(float64),
		P_internalFrag.Load().(float64),
		float64(AllocationTime.Load()) / 1_000_000_000,
		float64(DeallocationTime.Load()) / 1_000_000_000}
}
