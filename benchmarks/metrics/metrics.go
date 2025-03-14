package configurations

import "sync/atomic"

// Configurations
const (
	// Amount of goroutines
	Goroutines = 15

	// mat-mul
	Rows = 100
	Cols = 100

	//bin-tree
	BinOp = 1000

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

type Metrics struct {
	ComputationTime  float64
	Throughput       float64
	Latency          float64
	AllocationTime   float64
	DeallocationTime float64
}
