package gc

import (
	. "experiments/benchmarks/metrics"
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
	
	var obj *[32]byte
	for i := 0; i <  numAllocations; i++ {
		allocationTimeStart := time.Now()
		obj = new([32]byte)
		AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())
		_ = obj
	}

	deallocationStart := time.Now()
	runtime.GC()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	ComputationTime.Store(time.Since(computationTimeStart).Nanoseconds())

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

	allocationTimeStart := time.Now()
	done := make(chan bool)
	jobs := make(chan payload, Goroutines)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	for i := 0; i < Goroutines; i++ {
		go func() {
			for job := range jobs {
				Latency.Add(time.Since(job.latencyStart).Nanoseconds())
			}
			done <- true
		}()
	}

	var obj *payload
	for i := 0; i < Goroutines * numMessages; i++ {
		allocationTimeStart := time.Now()
		obj = new(payload)
		AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

		obj.latencyStart = time.Now()
		jobs <- *obj
	}

	close(jobs)

	for i := 0; i < Goroutines; i++ {
		<-done
	}

	deallocationStart := time.Now()
	runtime.GC()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	ComputationTime.Store(time.Since(computationTimeStart).Nanoseconds())

	return SystemMetrics{
		float64(ComputationTime.Load()),
		float64(Goroutines * numMessages) / float64(ComputationTime.Load()),
		float64(Latency.Load()),
		float64(AllocationTime.Load()),
		float64(DeallocationTime.Load())}
}