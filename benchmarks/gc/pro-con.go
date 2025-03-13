package gc

import (
	. "experiments/benchmarks/metrics"
	"math/rand/v2"
	"runtime"
	"runtime/debug"
	"time"
)

func producing(buffer chan int, done chan bool, op int, valueRange int) {
	for i := new(int); *i < op; *i++ {
		buffer <- rand.IntN(valueRange)
	}
	done <- true
}

func consuming(buffer chan int, done chan bool) {
	allocationStart := time.Now()
	x := new(int)
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

		internalFrag := float64(memStats.HeapIntFrag) / float64(memStats.HeapAlloc)
		if internalFrag > P_internalFrag.Load().(float64) {
			P_internalFrag.Store(internalFrag)
		}
	}
	done <- true
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

	allocationStart := time.Now()
	buffer := make(chan int)
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
		float64(ComputationTime.Load()) / 1_000_000_000,
		float64(ProConOp*1_000_000_000) / float64(ComputationTime.Load()),
		float64(Latency.Load()) / 1_000_000_000,
		float64(P_memoryConsuption.Load()),
		P_externalFrag.Load().(float64),
		P_internalFrag.Load().(float64),
		float64(AllocationTime.Load()) / 1_000_000_000,
		float64(DeallocationTime.Load()) / 1_000_000_000}
}
