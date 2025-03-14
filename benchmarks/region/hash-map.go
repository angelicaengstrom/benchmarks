//go:build goexperiment.regions

package region

import (
	. "experiments/benchmarks/metrics"
	"math/rand/v2"
	"region"
	"runtime/debug"
	"time"
)

type list struct {
	value int
	next  *list
}

func (l *list) push(v int, r *region.Region) bool {
	if l.value == v {
		return false
	} else if l.value == 0 {
		l.value = v
		return true
	} else if l.next == nil {
		allocationTimeStart := time.Now()
		l.next = region.AllocFromRegion[list](r)
		AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())
		l.next.value = v
		return true
	}

	return l.next.push(v, r)
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
	buckets [HashCap]*bucket
	size    int
}

func (b bucket) add(value int, r *region.Region) bool {
	return b.store.push(value, r)
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

func NewFineGrainedMap(r *region.Region) *FineGrainedMap {
	allocationTimeStart := time.Now()
	buckets := region.AllocFromRegion[[HashCap]*bucket](r)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	for i := range buckets {
		allocationTimeStart = time.Now()
		(*buckets)[i] = region.AllocFromRegion[bucket](r)
		(*buckets)[i].store = region.AllocFromRegion[list](r)
		(*buckets)[i].requests = region.AllocChannel[request](0, r)
		AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

		r.IncRefCounter()
		go (*buckets)[i].run(r)
	}
	allocationTimeStart = time.Now()
	fgm := region.AllocFromRegion[FineGrainedMap](r)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	fgm.buckets = *buckets
	fgm.size = HashCap

	return fgm
}

func (b bucket) run(r *region.Region) {
	allocationTimeStart := time.Now()
	req := region.AllocFromRegion[request](r)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	for *req = range b.requests {
		Latency.Add(time.Since(req.latencyStart).Nanoseconds())
		switch req.op {
		case OpInsert:
			req.result <- b.add(req.value, r)
		case OpSearch:
			req.result <- b.search(req.value)
		case OpRemove:
			req.result <- b.remove(req.value)
		}

	}
	r.DecRefCounter()
}

func (m *FineGrainedMap) hashKey(key int) int {
	return key % m.size
}

func (m *FineGrainedMap) Insert(value int, r *region.Region) bool {
	allocationTimeStart := time.Now()
	idx := region.AllocFromRegion[int](r)
	res := region.AllocChannel[bool](0, r)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	*idx = m.hashKey(value)

	m.buckets[*idx].requests <- request{value: value, op: OpInsert, result: res, latencyStart: time.Now()}
	return <-res
}

func (m *FineGrainedMap) Search(value int, r *region.Region) bool {
	allocationTimeStart := time.Now()
	idx := region.AllocFromRegion[int](r)
	res := region.AllocChannel[bool](0, r)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	*idx = m.hashKey(value)

	m.buckets[*idx].requests <- request{value: value, op: OpSearch, result: res, latencyStart: time.Now()}
	return <-res
}

func (m *FineGrainedMap) Delete(value int, r *region.Region) bool {
	allocationTimeStart := time.Now()
	idx := region.AllocFromRegion[int](r)
	res := region.AllocChannel[bool](0, r)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	*idx = m.hashKey(value)
	m.buckets[*idx].requests <- request{value: value, op: OpRemove, result: res, latencyStart: time.Now()}
	return <-res
}

func generateHashMapOperations(m *FineGrainedMap, valueRange int, op int, done chan bool, r *region.Region) {
	for i := region.AllocFromRegion[int](r); *i < op; *i++ {
		val := rand.IntN(valueRange) + 1
		switch method := opType(rand.IntN(3)); method {
		case OpInsert:
			m.Insert(val, r)
		case OpSearch:
			m.Search(val, r)
		case OpRemove:
			m.Delete(val, r)
		}
	}
	done <- true
	r.DecRefCounter()
}

func RunHashMap(valueRange int) Metrics {
	debug.SetGCPercent(-1)

	ComputationTime.Store(0)
	AllocationTime.Store(0)
	DeallocationTime.Store(0)
	Latency.Store(0)

	computationTimeStart := time.Now()
	r1 := region.CreateRegion()

	allocationTimeStart := time.Now()
	done := region.AllocChannel[bool](0, r1)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	m := NewFineGrainedMap(r1)

	for i := 0; i < Goroutines; i++ {
		r1.IncRefCounter()
		go generateHashMapOperations(m, valueRange, HashOp, done, r1)
	}

	for i := 0; i < Goroutines; i++ {
		<-done
	}
	ComputationTime.Store(time.Since(computationTimeStart).Nanoseconds())

	deallocationStart := time.Now()
	r1.RemoveRegion()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	return Metrics{
		float64(ComputationTime.Load()) / 1_000,
		float64(HashOp*1_000_000) / float64(ComputationTime.Load()),
		float64(Latency.Load()) / 1_000,
		float64(AllocationTime.Load()) / 1_000,
		float64(DeallocationTime.Load()) / 1_000}
}
