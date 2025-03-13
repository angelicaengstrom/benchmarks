package configurations

import "sync/atomic"

// Configurations
const (
	// Amount of goroutines
	Goroutines = 5

	// mat-mul
	Rows = 100
	Cols = 100

	//bin-tree
	BinOp = 100

	//pro-con
	ProConOp = 10000

	//serv-hand
	ServHandOp = 100

	//hash-map
	HashOp    = 100
	HashRange = 100
	HashCap   = HashRange * 4 / 3
)

var ComputationTime atomic.Int64
var Throughput atomic.Int64
var Latency atomic.Int64
var AllocationTime atomic.Int64
var DeallocationTime atomic.Int64
var P_externalFrag atomic.Value
var P_internalFrag atomic.Value
var P_memoryConsuption atomic.Uint64

type Metrics struct {
	ComputationTime   float64
	Throughput        float64
	Latency           float64
	MemoryConsumption float64
	ExternalFrag      float64
	InternalFrag      float64
	AllocationTime    float64
	DeallocationTime  float64
}
