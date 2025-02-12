package test

import (
	"experiments/benchmarks"
	"runtime"
	"runtime/debug"
	"testing"
	"time"
)

func BenchmarkConcurrentBinaryTree2(b *testing.B) {
	n := 2
	v := 1000
	op := 10000
	rounds := 10
	var totalThroughput [3]float64
	var totalLatency [3]time.Duration
	var totalComputationTime [3]time.Duration
	var totalDeallocateTime time.Duration
	var totalOperations [3]int
	b.ResetTimer()
	for j := 0; j < rounds; j++ {
		b.StopTimer()
		debug.SetGCPercent(-1)
		latency, computationTime, throughput, operations := benchmarks.ConcurrentBinaryTree(n, v, op)

		deallocationTime := time.Now()
		debug.SetGCPercent(100)
		runtime.GC()
		totalDeallocateTime += time.Since(deallocationTime)
		b.StopTimer()
		for i := 0; i < 3; i++ {
			totalLatency[i] += latency[i]
			totalThroughput[i] += throughput[i]
			totalComputationTime[i] += computationTime[i]
			totalOperations[i] += operations[i]
		}
	}

	var avgThroughput [3]float64
	var avgLatency [3]time.Duration
	var avgComputationTime [3]time.Duration
	var avgOperations [3]int

	for i := 0; i < 3; i++ {
		avgThroughput[i] = totalThroughput[i] / float64(rounds)
		avgLatency[i] = totalLatency[i] / time.Duration(rounds)
		avgComputationTime[i] = totalComputationTime[i] / time.Duration(rounds)
		avgOperations[i] = totalOperations[i] / rounds
	}

	b.Log("operations: ", avgOperations)
	b.Log("computationTime: ", avgComputationTime)
	b.Log("latency: ", avgLatency)
	b.Log("throughput: ", avgThroughput)
	b.Log("deallocateTime: ", (totalDeallocateTime / time.Duration(rounds)).Seconds())
}

func BenchmarkConcurrentBinaryTree4(b *testing.B) {
	n := 4
	v := 1000
	op := 100000
	rounds := 10
	var totalThroughput [3]float64
	var totalLatency [3]time.Duration
	var totalComputationTime [3]time.Duration
	var totalDeallocateTime time.Duration
	var totalOperations [3]int
	b.ResetTimer()
	for j := 0; j < rounds; j++ {
		b.StopTimer()
		debug.SetGCPercent(-1)
		latency, computationTime, throughput, operations := benchmarks.ConcurrentBinaryTree(n, v, op)

		deallocationTime := time.Now()
		debug.SetGCPercent(100)
		runtime.GC()
		totalDeallocateTime += time.Since(deallocationTime)
		b.StopTimer()
		for i := 0; i < 3; i++ {
			totalLatency[i] += latency[i]
			totalThroughput[i] += throughput[i]
			totalComputationTime[i] += computationTime[i]
			totalOperations[i] += operations[i]
		}
	}

	var avgThroughput [3]float64
	var avgLatency [3]time.Duration
	var avgComputationTime [3]time.Duration
	var avgOperations [3]int

	for i := 0; i < 3; i++ {
		avgThroughput[i] = totalThroughput[i] / float64(rounds)
		avgLatency[i] = totalLatency[i] / time.Duration(rounds)
		avgComputationTime[i] = totalComputationTime[i] / time.Duration(rounds)
		avgOperations[i] = totalOperations[i] / rounds
	}

	b.Log("operations: ", avgOperations)
	b.Log("computationTime: ", avgComputationTime)
	b.Log("latency: ", avgLatency)
	b.Log("throughput: ", avgThroughput)
	b.Log("deallocateTime: ", (totalDeallocateTime / time.Duration(rounds)).Seconds())
}
