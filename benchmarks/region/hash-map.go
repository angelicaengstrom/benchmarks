//go:build goexperiment.regions

package region

import (
	. "experiments/benchmarks/metrics"
	"region"
	"runtime"
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
	done     chan bool
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
	fgm := region.AllocFromRegion[FineGrainedMap](r)
	i := region.AllocFromRegion[int](r)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	for *i = 0; *i < HashCap; *i++ {
		if r.IncRefCounter() {
			allocationTimeStart = time.Now()
			(*buckets)[*i] = region.AllocFromRegion[bucket](r)
			(*buckets)[*i].store = region.AllocFromRegion[list](r)
			(*buckets)[*i].requests = region.AllocChannel[request](0, r)
			(*buckets)[*i].done = region.AllocChannel[bool](0, r)
			AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())
			
			go (*buckets)[*i].run(r)
		}
	}

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
	b.done <- true
}

func (m *FineGrainedMap) hashKey(key int) int {
	return key % m.size
}

func (m *FineGrainedMap) Insert(value int, idx int, res chan bool, req request) bool {
	idx = m.hashKey(value)
	req.value = value
	req.op = OpInsert
	req.result = res
	req.latencyStart = time.Now()
	m.buckets[idx].requests <- req
	return <-res
}

func (m *FineGrainedMap) Search(value int, idx int, res chan bool) bool {
	idx = m.hashKey(value)
	m.buckets[idx].requests <- request{value: value, op: OpSearch, result: res, latencyStart: time.Now()}
	return <-res
}

func (m *FineGrainedMap) Delete(value int, idx int, res chan bool) bool {
	idx = m.hashKey(value)
	m.buckets[idx].requests <- request{value: value, op: OpRemove, result: res, latencyStart: time.Now()}
	return <-res
}

func closeBuckets(m *FineGrainedMap) {
	for i := range m.buckets {
		close(m.buckets[i].requests)
		<-m.buckets[i].done
	}
}

func generateHashMapOperations(m *FineGrainedMap, valueRange int, op int, done chan bool, r *region.Region) {
	r2 := region.CreateRegion(0)

	allocationTimeStart := time.Now()
	idx := region.AllocFromRegion[int](r2)
	res := region.AllocChannel[bool](0, r2)
	req := region.AllocFromRegion[request](r2)
	i := region.AllocFromRegion[int](r2)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	for *i = 0; *i < op; *i++ {
		m.Insert(*i + valueRange, *idx, res, *req)
		/*
		val := rand.IntN(valueRange) + 1
		switch method := opType(rand.IntN(3)); method {
		case OpInsert:
			m.Insert(val, *idx, res)
		case OpSearch:
			m.Search(val, *idx, res)
		case OpRemove:
			m.Delete(val, *idx, res)
		}*/
	}

	deallocationStart := time.Now()
	r2.RemoveRegion()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	r.DecRefCounter()
	done <- true
}

func RunHashMap() SystemMetrics {
	debug.SetGCPercent(-1)

	ComputationTime.Store(0)
	AllocationTime.Store(0)
	DeallocationTime.Store(0)
	Latency.Store(0)

	computationTimeStart := time.Now()
	r1 := region.CreateRegion(RegionBlockBytes / 6)

	allocationTimeStart := time.Now()
	done := region.AllocChannel[bool](0, r1)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	m := NewFineGrainedMap(r1)

	valueRange := 0
	for i := 0; i < Goroutines; i++ {
		if r1.IncRefCounter() {
			go generateHashMapOperations(m, valueRange, HashOp, done, r1)
		}
		valueRange += HashOp
	}

	for i := 0; i < Goroutines; i++ {
		<-done
	}

	closeBuckets(m)

	deallocationStart := time.Now()
	r1.RemoveRegion()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	ComputationTime.Store(time.Since(computationTimeStart).Nanoseconds())

	runtime.GC()

	return SystemMetrics{
		float64(ComputationTime.Load()),
		float64(HashOp * Goroutines) / float64(ComputationTime.Load()),
		float64(Latency.Load()),
		float64(AllocationTime.Load()),
		float64(DeallocationTime.Load())}
}
