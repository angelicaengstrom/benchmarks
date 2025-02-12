package main

import (
	"runtime"
	"runtime/debug"
	"testing"
	"time"
)

func BenchmarkConcurrentMatrixMultiplication2(b *testing.B) {
	rows := 500
	cols := 500
	n := 2
	A := generateMatrix[int](rows, cols)
	B := generateMatrix[int](cols, rows)
	var totalThroughput float64
	var totalLatency time.Duration
	var totalComputationTime time.Duration
	var totalDeallocateTime time.Duration
	b.ResetTimer()
	for j := 0; j < 10; j++ {
		debug.SetGCPercent(-1)
		_, latency, throughput, computationTime := concurrentMatrixMultiplication[int](A, B, n)
		totalThroughput += throughput
		totalLatency += latency
		totalComputationTime += computationTime
		deallocationTime := time.Now()
		debug.SetGCPercent(100)
		runtime.GC()
		totalDeallocateTime += time.Since(deallocationTime)
	}
	b.StopTimer()
	b.Log("computationTime: ", (totalComputationTime / time.Duration(10)).Seconds())
	b.Log("latency: ", (totalLatency / time.Duration(10)).Seconds())
	b.Log("throughput: ", totalThroughput/10.0)
	b.Log("deallocateTime: ", (totalDeallocateTime / time.Duration(10)).Seconds())
}

func BenchmarkConcurrentMatrixMultiplication4(b *testing.B) {
	rows := 500
	cols := 500
	n := 4
	A := generateMatrix[int](rows, cols)
	B := generateMatrix[int](cols, rows)
	var avgThroughput float64
	var avgLatency time.Duration
	var avgComputationTime time.Duration
	var totalDeallocateTime time.Duration
	b.ResetTimer()
	for j := 0; j < 10; j++ {
		debug.SetGCPercent(-1)
		_, latency, throughput, computationTime := concurrentMatrixMultiplication[int](A, B, n)
		avgThroughput += throughput
		avgLatency += latency
		avgComputationTime += computationTime
		deallocationTime := time.Now()
		debug.SetGCPercent(100)
		runtime.GC()
		totalDeallocateTime += time.Since(deallocationTime)
	}
	b.StopTimer()
	b.Log("computationTime: ", (avgComputationTime / time.Duration(10)).Seconds())
	b.Log("latency: ", (avgLatency / time.Duration(10)).Seconds())
	b.Log("throughput: ", avgThroughput/10.0)
	b.Log("deallocateTime: ", (totalDeallocateTime / time.Duration(10)).Seconds())
}
