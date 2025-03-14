package gc

import (
	. "experiments/benchmarks/metrics"
	"math/rand/v2"
	"runtime"
	"runtime/debug"
	"time"
)

type list struct {
	value int
	next  *list
}

func (l *list) push(v int) bool {
	if l.value == v {
		return false
	} else if l.value == 0 {
		l.value = v
		return true
	} else if l.next == nil {
		allocationTimeStart := time.Now()
		l.next = new(list)
		AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())
		l.next.value = v
		return true
	}

	return l.next.push(v)
}

func (l *list) valueShift() {
	if l.next != nil && l.next.value != 0 {
		l.value = l.next.value
		l.next.valueShift()
	} else {
		l.value = 0
	}
}

func (l *list) remove(v int) bool {
	if l.value != v {
		if l.next == nil {
			return false
		}
		return l.next.remove(v)
	}
	l.valueShift()
	return true
}

type bucket struct {
	store    *list
	requests chan request
}

type FineGrainedMap struct {
	buckets []*bucket
	size    int
}

func (b bucket) add(value int) bool {
	return b.store.push(value)
}

func (b bucket) search(value int) bool {
	for e := b.store; e != nil && e.value != 0; e = e.next {
		if value == e.value {
			return true
		}
	}
	return false
}

func (b bucket) remove(value int) bool {
	return b.store.remove(value)
}

func NewFineGrainedMap() *FineGrainedMap {
	allocationTimeStart := time.Now()
	buckets := make([]*bucket, HashCap)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	for i := range buckets {
		allocationTimeStart = time.Now()
		buckets[i] = new(bucket)
		buckets[i].store = new(list)
		buckets[i].requests = make(chan request)
		AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

		go buckets[i].run()
	}
	allocationTimeStart = time.Now()
	fgm := new(FineGrainedMap)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	fgm.buckets = buckets
	fgm.size = HashCap

	return fgm
}

func (b bucket) run() {
	allocationTimeStart := time.Now()
	req := new(request)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	for *req = range b.requests {
		Latency.Add(time.Since(req.latencyStart).Nanoseconds())
		switch req.op {
		case OpInsert:
			req.result <- b.add(req.value)
		case OpSearch:
			req.result <- b.search(req.value)
		case OpRemove:
			req.result <- b.remove(req.value)
		}

	}
}

func (m *FineGrainedMap) hashKey(key int) int {
	return key % m.size
}

func (m *FineGrainedMap) Insert(value int) bool {
	allocationTimeStart := time.Now()
	idx := new(int)
	res := make(chan bool)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	*idx = m.hashKey(value)

	m.buckets[*idx].requests <- request{value: value, op: OpInsert, result: res, latencyStart: time.Now()}
	return <-res
}

func (m *FineGrainedMap) Search(value int) bool {
	allocationTimeStart := time.Now()
	idx := new(int)
	res := make(chan bool)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	*idx = m.hashKey(value)

	m.buckets[*idx].requests <- request{value: value, op: OpSearch, result: res, latencyStart: time.Now()}
	return <-res
}

func (m *FineGrainedMap) Delete(value int) bool {
	allocationTimeStart := time.Now()
	idx := new(int)
	res := make(chan bool)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	*idx = m.hashKey(value)
	m.buckets[*idx].requests <- request{value: value, op: OpRemove, result: res, latencyStart: time.Now()}
	return <-res
}

func generateHashMapOperations(m *FineGrainedMap, valueRange int, op int, done chan bool) {
	for i := new(int); *i < op; *i++ {
		val := rand.IntN(valueRange) + 1
		switch method := opType(rand.IntN(3)); method {
		case OpInsert:
			m.Insert(val)
		case OpSearch:
			m.Search(val)
		case OpRemove:
			m.Delete(val)
		}
	}
	done <- true
}

func RunHashMap(valueRange int) Metrics {
	debug.SetGCPercent(-1)

	ComputationTime.Store(0)
	AllocationTime.Store(0)
	DeallocationTime.Store(0)
	Latency.Store(0)

	allocationTimeStart := time.Now()
	done := make(chan bool)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	computationTimeStart := time.Now()
	m := NewFineGrainedMap()

	for i := 0; i < Goroutines; i++ {
		go generateHashMapOperations(m, valueRange, HashOp, done)
	}

	for i := 0; i < Goroutines; i++ {
		<-done
	}
	ComputationTime.Store(time.Since(computationTimeStart).Nanoseconds())

	deallocationStart := time.Now()
	runtime.GC()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	return Metrics{
		float64(ComputationTime.Load()) / 1_000,
		float64(HashOp*1_000_000) / float64(ComputationTime.Load()),
		float64(Latency.Load()) / 1_000,
		float64(AllocationTime.Load()) / 1_000,
		float64(DeallocationTime.Load()) / 1_000}
}
