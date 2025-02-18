package benchmarks

import (
	"container/list"
	"fmt"
	"math/rand/v2"
)

type bucket struct {
	store    *list.List
	requests chan request
}

type FineGrainedMap struct {
	buckets []*bucket
	size    int
}

func (b *bucket) add(value int) bool {
	for e := b.store.Front(); e != nil; e = e.Next() {
		if value == e.Value.(int) {
			return false
		}
	}
	b.store.PushBack(value)
	return true
}

func (b *bucket) search(value int) bool {
	for e := b.store.Front(); e != nil; e = e.Next() {
		if value == e.Value.(int) {
			return true
		}
	}
	return false
}

func (b *bucket) remove(value int) bool {
	for e := b.store.Front(); e != nil; e = e.Next() {
		if value == e.Value.(int) {
			b.store.Remove(e)
			return true
		}
	}
	return false
}

func NewFineGrainedMap(sz int) *FineGrainedMap {
	buckets := make([]*bucket, sz)
	for i := range buckets {
		buckets[i] = &bucket{
			store:    &list.List{},
			requests: make(chan request),
		}
		go buckets[i].run()
	}
	return &FineGrainedMap{
		buckets: buckets,
		size:    sz,
	}
}

func (b *bucket) run() {
	for req := range b.requests {
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
	idx := m.hashKey(value)
	res := make(chan bool)
	m.buckets[idx].requests <- request{value: value, op: OpInsert, result: res}
	return <-res
}

func (m *FineGrainedMap) Search(value int) bool {
	idx := m.hashKey(value)
	res := make(chan bool)
	m.buckets[idx].requests <- request{value: value, op: OpSearch, result: res}
	return <-res
}

func (m *FineGrainedMap) Delete(value int) bool {
	idx := m.hashKey(value)
	res := make(chan bool)
	m.buckets[idx].requests <- request{value: value, op: OpRemove, result: res}
	return <-res
}

func generateHashMapOperations(m *FineGrainedMap, valueRange int, op int) {
	for i := 0; i < op; i++ {
		val := rand.IntN(valueRange)
		switch method := opType(rand.IntN(3)); method {
		case OpInsert:
			fmt.Println("Inserting ", val, m.Insert(val))
		case OpSearch:
			fmt.Println("Searching ", val, m.Search(val))
		case OpRemove:
			fmt.Println("Deleting ", val, m.Delete(val))
		}
	}
}

func RunHashMap(n int, valueRange int, op int) {
	m := NewFineGrainedMap(valueRange + 1)
	for i := 0; i < n; i++ {
		go generateHashMapOperations(m, valueRange, op)
	}
}
