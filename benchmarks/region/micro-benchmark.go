//go:build goexperiment.regions

package region

import (
	. "experiments/benchmarks/metrics"
	"region"
	"runtime"
	"runtime/debug"
	"time"
)

func RunAlloc() SystemMetrics {
	debug.SetGCPercent(-1)

	ComputationTime.Store(0)
	AllocationTime.Store(0)
	DeallocationTime.Store(0)
	Latency.Store(0)
	numAllocations := 250_000
	computationTimeStart := time.Now()
	
	r := region.CreateRegion(32 * numAllocations)
	
	for i := 0; i < numAllocations; i++ {
		allocationTimeStart := time.Now()
		_ = region.AllocFromRegion[[32]byte](r)
		AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())
	}

	deallocationStart := time.Now()
	r.RemoveRegion()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	ComputationTime.Store(time.Since(computationTimeStart).Nanoseconds())

	runtime.GC()
	return SystemMetrics{
		float64(ComputationTime.Load()),
		float64(numAllocations) / float64(ComputationTime.Load()),
		float64(Latency.Load()),
		float64(AllocationTime.Load()),
		float64(DeallocationTime.Load())}
}

type payload struct {
	latencyStart time.Time
	buf          [32]byte
}

func RunChannel() SystemMetrics {
	debug.SetGCPercent(-1)

	ComputationTime.Store(0)
	AllocationTime.Store(0)
	DeallocationTime.Store(0)
	Latency.Store(0)

	numMessages := 10000

	computationTimeStart := time.Now()
	r := region.CreateRegion(numMessages * Goroutines * 32)

	allocationTimeStart := time.Now()
	done := region.AllocChannel[bool](0, r)
	jobs := region.AllocChannel[payload](Goroutines, r)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	for i := 0; i < Goroutines; i++ {
		go func() {
			for job := range jobs {
				Latency.Add(time.Since(job.latencyStart).Nanoseconds())
			}
			done <- true
		}()
	}

	for i := 0; i < Goroutines * numMessages; i++ {
		allocationTimeStart := time.Now()
		obj := region.AllocFromRegion[payload](r)
		AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

		obj.latencyStart = time.Now()
		jobs <- *obj
	}

	close(jobs)

	for i := 0; i < Goroutines; i++ {
		<-done
	}

	deallocationStart := time.Now()
	r.RemoveRegion()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	ComputationTime.Store(time.Since(computationTimeStart).Nanoseconds())

	runtime.GC()
	return SystemMetrics{
		float64(ComputationTime.Load()),
		float64(Goroutines * numMessages) / float64(ComputationTime.Load()),
		float64(Latency.Load()),
		float64(AllocationTime.Load()),
		float64(DeallocationTime.Load())}
}