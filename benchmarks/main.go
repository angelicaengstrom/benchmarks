//go:build goexperiment.regions

package main

import (
	"encoding/csv"
	"experiments/benchmarks/gc"
	. "experiments/benchmarks/metrics"
	"experiments/benchmarks/region"
	"fmt"
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
	Program = "hash-map"
)

func main() {
	mm := MemoryManager(GC)
	var avgMetrics Metrics
	for i := 0; i < WarmUp; i++ {
		runTests(mm)
	}
	stop.Store(false)
	go measureMemStats(mm)
	for i := 0; i < Rounds; i++ {
		m := runTests(mm)
		avgMetrics = average(avgMetrics, m, float64(Rounds))
	}
	stop.Store(true)

	writeResults(avgMetrics, mm)
	fmt.Print(avgMetrics)
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
	avg.ExternalFrag += m.ExternalFrag / n
	avg.InternalFrag += m.InternalFrag / n
	avg.MemoryConsumption += m.MemoryConsumption / n
	avg.Latency += m.Latency / n
	avg.Throughput += m.Throughput / n
	return avg
}

func measureMemStats(mm MemoryManager) {
	var memStats runtime.MemStats
	var memCons, extFrag, intFrag float64
	var stamp int64
	start := time.Now()

	var data [][]string
	header := []string{"Time", "M_C", "ExtFrag", "IntFrag"}
	data = append(data, header)

	for !stop.Load() {
		runtime.ReadMemStats(&memStats)
		stamp = time.Since(start).Nanoseconds()
		memCons = float64(memStats.HeapAlloc)
		extFrag = float64(memStats.HeapIdle) / float64(memStats.HeapSys)
		if mm == GC {
			intFrag = float64(memStats.HeapIntFrag) / float64(memStats.HeapAlloc)
		} else {
			intFrag = float64(memStats.RegionIntFrag) / float64(memStats.RegionInUse)
		}

		data = append(data, []string{
			strconv.Itoa(int(stamp)),
			strconv.FormatFloat(memCons, 'f', -1, 64),
			strconv.FormatFloat(extFrag, 'f', -1, 64),
			strconv.FormatFloat(intFrag, 'f', -1, 64)})
	}

	file, _ := os.OpenFile("results/"+Program+"-mem.csv", os.O_WRONLY|os.O_CREATE, 0600)
	csvWriter := csv.NewWriter(file)
	csvWriter.WriteAll(data)
	file.Close()
}

func writeResults(metrics Metrics, mm MemoryManager) {
	var mmStr string
	switch mm {
	case GC:
		mmStr = "GC"
	case RBMM:
		mmStr = "RBMM"
	}
	file, _ := os.OpenFile("results/"+Program+".csv", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)

	csvWriter := csv.NewWriter(file)

	var output [][]string
	var header []string
	var headerData []string
	switch Program {
	case "mat-mul":
		header = []string{"Program", "MemMan", "Goroutines", "WarmUps", "Rounds", "ValRange", "Operations"}
		headerData = []string{Program, mmStr, strconv.Itoa(Goroutines), strconv.Itoa(WarmUp), strconv.Itoa(Rounds), strconv.Itoa(Range), strconv.Itoa(Rows * Cols)}
	case "bin-tree":
		header = []string{"Program", "MemMan", "Goroutines", "WarmUps", "Rounds", "ValRange", "Operations"}
		headerData = []string{Program, mmStr, strconv.Itoa(Goroutines), strconv.Itoa(WarmUp), strconv.Itoa(Rounds), strconv.Itoa(Range), strconv.Itoa(BinOp * Goroutines)}
	case "pro-con":
		header = []string{"Program", "MemMan", "Goroutines", "WarmUps", "Rounds", "ValRange", "Operations"}
		headerData = []string{Program, mmStr, strconv.Itoa(Goroutines * 2), strconv.Itoa(WarmUp), strconv.Itoa(Rounds), strconv.Itoa(Range), strconv.Itoa(ProConOp)}
	case "serv-hand":
		header = []string{"Program", "MemMan", "Goroutines", "WarmUps", "Rounds", "Operations"}
		headerData = []string{Program, mmStr, strconv.Itoa(Goroutines), strconv.Itoa(WarmUp), strconv.Itoa(Rounds), strconv.Itoa(ServHandOp)}
	case "hash-map":
		header = []string{"Program", "MemMan", "Goroutines", "WarmUps", "Rounds", "ValRange", "Operations"}
		headerData = []string{Program, mmStr, strconv.Itoa(Goroutines), strconv.Itoa(WarmUp), strconv.Itoa(Rounds), strconv.Itoa(HashRange), strconv.Itoa(HashOp * Goroutines)}
	default:
		header = []string{"Program", "MemMan", "Goroutines", "WarmUps", "Rounds"}
		headerData = []string{Program, mmStr, strconv.Itoa(Goroutines), strconv.Itoa(WarmUp), strconv.Itoa(Rounds)}
	}
	output = append(output, header)
	output = append(output, headerData)

	metricsHeader := []string{"T_C", "T_L", "Theta", "M_C", "M_E", "M_F", "T_A", "T_D"}
	metricsData := []string{
		strconv.FormatFloat(metrics.ComputationTime, 'f', -1, 64),
		strconv.FormatFloat(metrics.Latency, 'f', -1, 64),
		strconv.FormatFloat(metrics.Throughput, 'f', -1, 64),
		strconv.FormatFloat(metrics.MemoryConsumption, 'f', -1, 64),
		strconv.FormatFloat(metrics.ExternalFrag, 'f', -1, 64),
		strconv.FormatFloat(metrics.InternalFrag, 'f', -1, 64),
		strconv.FormatFloat(metrics.AllocationTime, 'f', -1, 64),
		strconv.FormatFloat(metrics.DeallocationTime, 'f', -1, 64)}

	output = append(output, metricsHeader)
	output = append(output, metricsData)

	csvWriter.WriteAll(output)

	file.Close()
}
