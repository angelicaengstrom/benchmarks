//go:build goexperiment.regions

package main

import (
	"encoding/csv"
	"experiments/benchmarks/gc"
	. "experiments/benchmarks/metrics"
	"experiments/benchmarks/region"
	"os"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"
)

type MemoryManager int

var stop atomic.Bool

const (
	GC      = MemoryManager(iota)
	RBMM    = MemoryManager(iota)
	WarmUp  = 5
	Rounds  = 10
	Range   = 100
	Program = "bin-tree"
)

func main() {
	mm := MemoryManager(RBMM)
	var avgMetrics Metrics
	done := make(chan bool)

	for i := 0; i < WarmUp; i++ {
		runTests(mm)
	}
	// Measure memory
	stop.Store(false)
	go measureMemStats(mm, done)
	for i := 0; i < Rounds; i++ {
		m := runTests(mm)
		avgMetrics = average(avgMetrics, m, float64(Rounds))
	}
	stop.Store(true)

	measureSysStats(avgMetrics, mm)
	<-done
}

func runTests(mm MemoryManager) Metrics {
	var m Metrics
	switch mm {
	case GC:
		switch Program {
		case "mat-mul":
			m = gc.RunMatrixMultiplication(Range)
		case "bin-tree":
			m = gc.RunBinaryTree(Range, BinOp)
		case "pro-con":
			m = gc.RunProducerConsumer(Range)
		case "serv-hand":
			m = gc.RunServerHandler()
		case "hash-map":
			m = gc.RunHashMap(HashRange)
		default:
			panic("unreachable")
		}
	case RBMM:
		switch Program {
		case "mat-mul":
			m = region.RunMatrixMultiplication(Range)
		case "bin-tree":
			m = region.RunBinaryTree(Range, BinOp)
		case "pro-con":
			m = region.RunProducerConsumer(Range)
		case "serv-hand":
			m = region.RunServerHandler()
		case "hash-map":
			m = region.RunHashMap(HashRange)
		default:
			panic("unreachable")
		}
	}
	return m
}

func average(avg Metrics, m Metrics, n float64) Metrics {
	avg.ComputationTime += m.ComputationTime / n
	avg.AllocationTime += m.AllocationTime / n
	avg.DeallocationTime += m.DeallocationTime / n
	avg.Latency += m.Latency / n
	avg.Throughput += m.Throughput / n
	return avg
}

func measureMemStats(mm MemoryManager, done chan bool) {
	var memStats runtime.MemStats
	var memCons, extFrag, intFrag, memReg int64
	var stamp int64
	start := time.Now()

	var data [][]string
	header := []string{"Time", "M_C", "ExtFrag", "IntFrag", "M_R"}
	data = append(data, header)

	for !stop.Load() {
		runtime.ReadMemStats(&memStats)
		stamp = time.Since(start).Nanoseconds() / 1_000_000    // ms
		memCons = int64(memStats.HeapAlloc) / int64(1024*1024) // MB
		extFrag = int64(memStats.HeapIdle) / int64(1024*1024)  // MB
		switch mm {
		case GC:
			intFrag = int64(memStats.HeapIntFrag) / int64(1024*1024)
		case RBMM:
			if memStats.RegionIntFrag < uint64(^uint32(0)) {
				intFrag = int64(memStats.RegionIntFrag) / int64(1024*1024)
			}
		}
		memReg = int64(memStats.RegionInUse) / int64(1024*1024)

		data = append(data, []string{
			strconv.Itoa(int(stamp)),
			strconv.Itoa(int(memCons)),
			strconv.Itoa(int(extFrag)),
			strconv.Itoa(int(intFrag)),
			strconv.Itoa(int(memReg))})

		time.Sleep(time.Nanosecond * 1_000)
	}
	var mmStr string
	switch mm {
	case GC:
		mmStr = "GC"
	case RBMM:
		mmStr = "RBMM"
	}
	file, _ := os.OpenFile("results/"+Program+"-"+mmStr+"-mem.csv", os.O_WRONLY|os.O_CREATE, 0600)
	csvWriter := csv.NewWriter(file)
	csvWriter.WriteAll(data)
	file.Close()
	done <- true
}

func measureSysStats(metrics Metrics, mm MemoryManager) {
	var mmStr string
	switch mm {
	case GC:
		mmStr = "GC"
	case RBMM:
		mmStr = "RBMM"
	}

	var output [][]string
	if _, err := os.Stat("results/" + Program + "-" + mmStr + "-sys.csv"); os.IsNotExist(err) {
		metricsHeader := []string{"G", "WarmUps", "Rounds", "Operations", "T_C", "T_L", "Theta", "T_A", "T_D"}
		output = append(output, metricsHeader)
	}

	file, _ := os.OpenFile("results/"+Program+"-"+mmStr+"-sys.csv", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	csvWriter := csv.NewWriter(file)

	var metricsData []string
	switch Program {
	case "mat-mul":
		metricsData = []string{
			strconv.Itoa(Goroutines),
			strconv.Itoa(WarmUp),
			strconv.Itoa(Rounds),
			strconv.Itoa(Rows * Cols),
			strconv.Itoa(int(metrics.ComputationTime)),
			strconv.Itoa(int(metrics.Latency)),
			strconv.Itoa(int(metrics.Throughput)),
			strconv.Itoa(int(metrics.AllocationTime)),
			strconv.Itoa(int(metrics.DeallocationTime))}
	case "bin-tree":
		metricsData = []string{
			strconv.Itoa(Goroutines),
			strconv.Itoa(WarmUp),
			strconv.Itoa(Rounds),
			strconv.Itoa(BinOp * Goroutines),
			strconv.Itoa(int(metrics.ComputationTime)),
			strconv.Itoa(int(metrics.Latency)),
			strconv.Itoa(int(metrics.Throughput)),
			strconv.Itoa(int(metrics.AllocationTime)),
			strconv.Itoa(int(metrics.DeallocationTime))}
	case "pro-con":
		metricsData = []string{
			strconv.Itoa(Goroutines),
			strconv.Itoa(WarmUp),
			strconv.Itoa(Rounds),
			strconv.Itoa(ProConOp),
			strconv.Itoa(int(metrics.ComputationTime)),
			strconv.Itoa(int(metrics.Latency)),
			strconv.Itoa(int(metrics.Throughput)),
			strconv.Itoa(int(metrics.AllocationTime)),
			strconv.Itoa(int(metrics.DeallocationTime))}
	case "serv-hand":
		metricsData = []string{
			strconv.Itoa(Goroutines),
			strconv.Itoa(WarmUp),
			strconv.Itoa(Rounds),
			strconv.Itoa(ServHandOp),
			strconv.Itoa(int(metrics.ComputationTime)),
			strconv.Itoa(int(metrics.Latency)),
			strconv.Itoa(int(metrics.Throughput)),
			strconv.Itoa(int(metrics.AllocationTime)),
			strconv.Itoa(int(metrics.DeallocationTime))}
	case "hash-map":
		metricsData = []string{
			strconv.Itoa(Goroutines),
			strconv.Itoa(WarmUp),
			strconv.Itoa(Rounds),
			strconv.Itoa(HashOp * Goroutines),
			strconv.Itoa(int(metrics.ComputationTime)),
			strconv.Itoa(int(metrics.Latency)),
			strconv.Itoa(int(metrics.Throughput)),
			strconv.Itoa(int(metrics.AllocationTime)),
			strconv.Itoa(int(metrics.DeallocationTime))}
	}

	output = append(output, metricsData)

	csvWriter.WriteAll(output)

	file.Close()
}
