//go:build goexperiment.regions

package main

import (
	"encoding/csv"
	"experiments/benchmarks/gc"
	. "experiments/benchmarks/metrics"
	"experiments/benchmarks/region"
	"math"
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
	Program = "serv-hand"
)

func main() {
	mm := MemoryManager(RBMM)
	var sysMetrics [Rounds]SystemMetrics
	done := make(chan bool)

	for i := 0; i < WarmUp; i++ {
		runTests(mm)
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	stop.Store(false)
	go measureAllMemStats(mm, done, memStats)
	for i := 0; i < Rounds; i++ {
		sysMetrics[i] = runTests(mm)
	}
	stop.Store(true)

	avgSysMetrics := averageSysMetrics(sysMetrics)
	stdErrSysMetrics := stdErr(avgSysMetrics, sysMetrics, float64(Rounds))

	writeSysStats(avgSysMetrics, stdErrSysMetrics, mm)
	<-done
}

func runTests(mm MemoryManager) SystemMetrics {
	var m SystemMetrics
	switch mm {
	case GC:
		switch Program {
		case "mat-mul":
			m = gc.RunMatrixMultiplication(Range)
		case "bin-tree":
			m = gc.RunBinaryTree(BinOp)
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
			m = region.RunBinaryTree(BinOp)
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

func averageSysMetrics(m [Rounds]SystemMetrics) SystemMetrics {
	var avg SystemMetrics
	for i := 0; i < Rounds; i++ {
		avg.ComputationTime += m[i].ComputationTime / Rounds
		avg.AllocationTime += m[i].AllocationTime / Rounds
		avg.DeallocationTime += m[i].DeallocationTime / Rounds
		avg.Latency += m[i].Latency / Rounds
		avg.Throughput += m[i].Throughput / Rounds
	}

	return avg
}

func stdErr(mean SystemMetrics, metrics [Rounds]SystemMetrics, n float64) SystemMetrics {
	var sumSq SystemMetrics
	for _, m := range metrics {
		sumSq.ComputationTime += math.Pow(m.ComputationTime-mean.ComputationTime, 2)
		sumSq.AllocationTime += math.Pow(m.AllocationTime-mean.AllocationTime, 2)
		sumSq.DeallocationTime += math.Pow(m.DeallocationTime-mean.DeallocationTime, 2)
		sumSq.Latency += math.Pow(m.Latency-mean.Latency, 2)
		sumSq.Throughput += math.Pow(m.Throughput-mean.Throughput, 2)
	}
	stddev := SystemMetrics{
		ComputationTime:  math.Sqrt(sumSq.ComputationTime / (n - 1)),
		AllocationTime:   math.Sqrt(sumSq.AllocationTime / (n - 1)),
		DeallocationTime: math.Sqrt(sumSq.DeallocationTime / (n - 1)),
		Latency:          math.Sqrt(sumSq.Latency / (n - 1)),
		Throughput:       math.Sqrt(sumSq.Throughput / (n - 1)),
	}
	return SystemMetrics{
		ComputationTime:  stddev.ComputationTime / math.Sqrt(n),
		AllocationTime:   stddev.AllocationTime / math.Sqrt(n),
		DeallocationTime: stddev.DeallocationTime / math.Sqrt(n),
		Latency:          stddev.Latency / math.Sqrt(n),
		Throughput:       stddev.Throughput / math.Sqrt(n),
	}
}

func measureAllMemStats(mm MemoryManager, done chan bool, memStats runtime.MemStats) {
	var memCons, extFrag, intFrag float64
	var stamp, oldStamp int64

	var data [][]string
	header := []string{"Time", "M_C", "ExtFrag", "IntFrag"}
	data = append(data, header)

	memConsBefore := memStats.HeapAlloc
	intFragBefore := memStats.HeapIntFrag
	start := time.Now()
	for !stop.Load() {
		stamp = time.Since(start).Nanoseconds() / 1_000_000 // ms
		if oldStamp != stamp {

			runtime.ReadMemStats(&memStats)

			extFrag = float64(memStats.HeapIdle) / float64(1024*1024)                // MB
			memCons = float64(memStats.HeapAlloc-memConsBefore) / float64(1024*1024) // MB
			switch mm {
			case GC:
				intFrag = float64(memStats.HeapIntFrag-intFragBefore) / float64(1024*1024)
			case RBMM:
				if memStats.RegionIntFrag < uint64(^uint32(0)) {
					intFrag = float64(memStats.RegionIntFrag) / float64(1024*1024)
				}
			}

			data = append(data, []string{
				strconv.Itoa(int(stamp)),
				strconv.FormatFloat(memCons, 'f', 2, 64),
				strconv.FormatFloat(extFrag, 'f', 2, 64),
				strconv.FormatFloat(intFrag, 'f', 2, 64)})
		}
		oldStamp = stamp
	}
	var mmStr string
	switch mm {
	case GC:
		mmStr = "GC"
	case RBMM:
		mmStr = "RBMM"
	}
	file, _ := os.OpenFile("results/"+Program+"/"+strconv.Itoa(Goroutines)+"-"+mmStr+"-mem.csv", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	csvWriter := csv.NewWriter(file)
	csvWriter.WriteAll(data)
	file.Close()
	done <- true
}

func writeSysStats(avgMetrics SystemMetrics, stdErrMetrics SystemMetrics, mm MemoryManager) {
	var mmStr string
	switch mm {
	case GC:
		mmStr = "GC"
	case RBMM:
		mmStr = "RBMM"
	}

	var output [][]string
	if _, err := os.Stat("results/" + Program + "/" + mmStr + "-sys.csv"); os.IsNotExist(err) {
		metricsHeader := []string{"G", "T_C", "T_L", "Theta", "T_A", "T_D", "T_C_ERR", "T_L_ERR", "Theta_ERR", "T_A_ERR", "T_D_ERR"}
		output = append(output, metricsHeader)
	}

	file, _ := os.OpenFile("results/"+Program+"/"+mmStr+"-sys.csv", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	csvWriter := csv.NewWriter(file)

	metricsData := []string{
		strconv.Itoa(Goroutines),
		strconv.FormatFloat(avgMetrics.ComputationTime/1_000_000, 'f', 2, 64),
		strconv.FormatFloat(avgMetrics.Latency/1_000_000, 'f', 2, 64),
		strconv.FormatFloat(avgMetrics.Throughput*1_000_000, 'f', 2, 64),
		strconv.FormatFloat(avgMetrics.AllocationTime/1_000_000, 'f', 2, 64),
		strconv.FormatFloat(avgMetrics.DeallocationTime/1_000_000, 'f', 2, 64),
		strconv.FormatFloat(stdErrMetrics.ComputationTime/1_000_000, 'f', 2, 64),
		strconv.FormatFloat(stdErrMetrics.Latency/1_000_000, 'f', 2, 64),
		strconv.FormatFloat(stdErrMetrics.Throughput*1_000_000, 'f', 2, 64),
		strconv.FormatFloat(stdErrMetrics.AllocationTime/1_000_000, 'f', 2, 64),
		strconv.FormatFloat(stdErrMetrics.DeallocationTime/1_000_000, 'f', 2, 64),
	}

	output = append(output, metricsData)

	csvWriter.WriteAll(output)

	file.Close()
}

/*
func averageMemMetrics(m [Rounds][TimeLimit]MemoryMetrics) [TimeLimit]MemoryMetrics {
	var avg [TimeLimit]MemoryMetrics
	for i := 0; i < TimeLimit; i++ {
		avg[i].TimeStamp = float64(i)
		n := Rounds
		for j := 0; j < Rounds; j++ {
			if m[j][i].TimeStamp != float64(i) {
				n--
			}
			avg[i].MemoryConsumption += m[j][i].MemoryConsumption
			avg[i].InternalFrag += m[j][i].InternalFrag
			avg[i].ExternalFrag += m[j][i].ExternalFrag
			avg[i].MemoryRegion += m[j][i].MemoryRegion
		}
		if n != 0 {
			avg[i].MemoryConsumption /= float64(n)
			avg[i].InternalFrag /= float64(n)
			avg[i].ExternalFrag /= float64(n)
			avg[i].MemoryRegion /= float64(n)
		}
	}
	return avg
}

func stdErrMem(mean [TimeLimit]MemoryMetrics, metrics [Rounds][TimeLimit]MemoryMetrics) [TimeLimit]MemoryMetrics {
	var sumSq [TimeLimit]MemoryMetrics
	var stddev [TimeLimit]MemoryMetrics
	var stdMemErr [TimeLimit]MemoryMetrics
	for i := 0; i < TimeLimit; i++ {
		sumSq[i].TimeStamp = float64(i)
		n := float64(Rounds)
		for _, m := range metrics {
			if m[i].TimeStamp == float64(i) {
				sumSq[i].MemoryConsumption += math.Pow(m[i].MemoryConsumption-mean[i].MemoryConsumption, 2)
				sumSq[i].ExternalFrag += math.Pow(m[i].ExternalFrag-mean[i].ExternalFrag, 2)
				sumSq[i].InternalFrag += math.Pow(m[i].InternalFrag-mean[i].InternalFrag, 2)
				sumSq[i].MemoryRegion += math.Pow(m[i].MemoryRegion-mean[i].MemoryRegion, 2)
			} else {
				n--
			}
		}
		if n-1 != 0 {
			stddev[i] = MemoryMetrics{
				TimeStamp:         float64(i),
				MemoryConsumption: math.Sqrt(sumSq[i].MemoryConsumption / (n - 1)),
				ExternalFrag:      math.Sqrt(sumSq[i].ExternalFrag / (n - 1)),
				InternalFrag:      math.Sqrt(sumSq[i].InternalFrag / (n - 1)),
				MemoryRegion:      math.Sqrt(sumSq[i].MemoryRegion / (n - 1)),
			}
		}

		if math.Sqrt(n) != 0 {
			stdMemErr[i] = MemoryMetrics{
				TimeStamp:         float64(i),
				MemoryConsumption: stddev[i].MemoryConsumption / math.Sqrt(n),
				ExternalFrag:      stddev[i].ExternalFrag / math.Sqrt(n),
				InternalFrag:      stddev[i].InternalFrag / math.Sqrt(n),
				MemoryRegion:      stddev[i].MemoryRegion / math.Sqrt(n),
			}
		}
	}

	return stdMemErr
}

func measureMemStats(metrics *[TimeLimit]MemoryMetrics, done chan bool, mm MemoryManager, maxTime *int) {
	var memStats runtime.MemStats
	var memCons, extFrag, intFrag, memReg int64
	var stamp int64
	var oldStamp int64

	runtime.ReadMemStats(&memStats)
	memConsBefore := int64(memStats.HeapAlloc)
	start := time.Now()
	for !stop.Load() {
		stamp = time.Since(start).Nanoseconds() / 1_000_000 // ms
		if oldStamp != stamp {
			runtime.ReadMemStats(&memStats)
			extFrag = int64(memStats.HeapIdle) / int64(1024*1024) // MB
			switch mm {
			case GC:
				intFrag = int64(memStats.HeapIntFrag) / int64(1024*1024)
				memCons = (int64(memStats.HeapAlloc) - memConsBefore) / int64(1024*1024) // MB
			case RBMM:
				if memStats.RegionIntFrag < uint64(^uint32(0)) {
					intFrag = int64(memStats.RegionIntFrag) / int64(1024*1024)
				}
				memCons = int64(memStats.RegionInUse) / int64(1024*1024)
			}
			memReg = int64(memStats.RegionInUse) / int64(1024*1024)

			metrics[stamp] = MemoryMetrics{float64(stamp), float64(memCons), float64(extFrag), float64(intFrag), float64(memReg)}
			oldStamp = stamp
		}
		if *maxTime < int(stamp) {
			*maxTime = int(stamp)
		}
		time.Sleep(time.Nanosecond * 1000)
	}

	done <- true
}

func writeMemStats(avgMetrics [TimeLimit]MemoryMetrics, stdErrMetrics [TimeLimit]MemoryMetrics, mm MemoryManager, maxTime int) {
	var data [][]string
	header := []string{"Time", "M_C", "ExtFrag", "IntFrag", "M_R", "M_C_ERR", "ExtFrag_ERR", "IntFrag_ERR", "M_R_ERR"}
	data = append(data, header)
	for i := 0; i < maxTime; i++ {
		data = append(data, []string{
			strconv.Itoa(int(avgMetrics[i].TimeStamp)),
			strconv.Itoa(int(avgMetrics[i].MemoryConsumption)),
			strconv.Itoa(int(avgMetrics[i].ExternalFrag)),
			strconv.Itoa(int(avgMetrics[i].InternalFrag)),
			strconv.Itoa(int(avgMetrics[i].MemoryRegion)),
			strconv.Itoa(int(stdErrMetrics[i].MemoryConsumption)),
			strconv.Itoa(int(stdErrMetrics[i].ExternalFrag)),
			strconv.Itoa(int(stdErrMetrics[i].InternalFrag)),
			strconv.Itoa(int(stdErrMetrics[i].MemoryRegion))})
	}
	var mmStr string
	switch mm {
	case GC:
		mmStr = "GC"
	case RBMM:
		mmStr = "RBMM"
	}
	file, _ := os.OpenFile("results/"+Program+"/"+strconv.Itoa(Goroutines)+"-"+mmStr+"-mem.csv", os.O_WRONLY|os.O_CREATE, 0600)
	csvWriter := csv.NewWriter(file)
	csvWriter.WriteAll(data)
	file.Close()
}
*/
