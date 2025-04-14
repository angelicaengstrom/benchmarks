package configurations

import (
	"sync/atomic"
	"unsafe"
)

// Configurations
const (
	RegionBlockBytes = 8388608
	// Amount of goroutines
	Goroutines = 256

	// mat-mul
	Rows = 100 * (1 + (Goroutines >> 4))
	Cols = Rows

	//bin-tree
	BinOp    = 2000
	BinRange = BinOp * Goroutines

	//pro-con
	ProConOp = 10000

	//serv-hand
	ServHandOp = 100

	//hash-map
	HashOp    = 2000
	HashRange = HashOp
	HashCap   = (HashRange * Goroutines * 4) / 3
)

var ComputationTime atomic.Int64
var Throughput atomic.Int64
var Latency atomic.Int64
var AllocationTime atomic.Int64
var DeallocationTime atomic.Int64

type SystemMetrics struct {
	ComputationTime  float64
	Throughput       float64
	Latency          float64
	AllocationTime   float64
	DeallocationTime float64
}

type MemoryMetrics struct {
	TimeStamp         float64
	MemoryConsumption float64
	ExternalFrag      float64
	InternalFrag      float64
	MemoryRegion      float64
}

func New[T any]() *T {
	var x T
	return (*T)(unsafe.Pointer(&make([]T, unsafe.Sizeof(x))[0]))
}
