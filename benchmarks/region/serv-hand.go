//go:build goexperiment.regions

package region

import (
	. "experiments/benchmarks/metrics"
	"fmt"
	"net"
	"region"
	"runtime"
	"runtime/debug"
	"time"
)

type Request struct {
	conn         net.Conn
	latencyStart time.Time
	buf          [1024]byte
}

type server struct {
	requests chan Request
	listener net.Listener
}

func newServer(address string, r *region.Region) (*server, error) {
	allocationTimeStart := time.Now()
	s := region.AllocFromRegion[server](r)
	s.requests = region.AllocChannel[Request](0, r)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	s.listener, _ = net.Listen("tcp", address)

	return s, nil
}

func (s *server) acceptConnections(done chan bool, r1 *region.Region) {
	r2 := region.CreateRegion(ServHandOp * 1064 * Goroutines)
	for i := region.AllocFromRegion[int](r2); *i < ServHandOp*Goroutines; *i++ {
		allocationTimeStart := time.Now()
		req := region.AllocFromRegion[Request](r2)
		AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

		conn, err := s.listener.Accept()

		if err != nil {
			continue
		}

		req.conn = conn
		req.latencyStart = time.Now()
		s.requests <- *req
	}

	close(s.requests)

	deallocationStart := time.Now()
	r2.RemoveRegion()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	r1.DecRefCounter()
	done <- true
}

func (s *server) handleConnections(done chan bool, r1 *region.Region) {
	allocationTimeStart := time.Now()
	req := region.AllocFromRegion[Request](r1)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	for *req = range s.requests {
		Latency.Add(time.Since(req.latencyStart).Nanoseconds())
		s.handleConnection(*req)
	}
	r1.DecRefCounter()
	done <- true
}

func (s *server) handleConnection(req Request) {
	req.conn.Close()
}

func (s *server) run(done chan bool, r1 *region.Region) {
	r1.IncRefCounter()
	go s.acceptConnections(done, r1)

	r1.IncRefCounter()
	go s.handleConnections(done, r1)
}

func (s *server) stop(done chan bool) error {
	<-done // acceptConnections
	<-done // handleConnections
	return s.listener.Close()
}

func sendRequests(op int, done chan bool, address string, r1 *region.Region) {
	r2 := region.CreateRegion(op * 1064)
	for i := region.AllocFromRegion[int](r2); *i < op; *i++ {
		allocationTimeStart := time.Now()
		req := region.AllocFromRegion[Request](r2)
		AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

		req.conn, _ = net.Dial("tcp", address)

		req.conn.Close()
	}

	deallocationStart := time.Now()
	r2.RemoveRegion()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	r1.DecRefCounter()
	done <- true
}

func RunServerHandler() SystemMetrics {
	debug.SetGCPercent(-1)

	ComputationTime.Store(0)
	AllocationTime.Store(0)
	DeallocationTime.Store(0)
	Latency.Store(0)

	computationTimeStart := time.Now()

	r1 := region.CreateRegion(0)

	allocationTimeStart := time.Now()
	address := region.AllocFromRegion[string](r1)
	done := region.AllocChannel[bool](0, r1)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())
	*address = ":8080"

	s, err := newServer(*address, r1)
	if err != nil {
		fmt.Println(err)
		return SystemMetrics{}
	}

	s.run(done, r1)

	for i := 0; i < Goroutines; i++ {
		if r1.IncRefCounter() {
			go sendRequests(ServHandOp, done, *address, r1)
		}
	}

	for i := 0; i < Goroutines; i++ {
		<-done
	}

	if s.stop(done) != nil {
		fmt.Println("Could not stop server")
	}

	deallocationStart := time.Now()
	r1.RemoveRegion()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	ComputationTime.Store(time.Since(computationTimeStart).Nanoseconds())

	runtime.GC()
	return SystemMetrics{
		float64(ComputationTime.Load()),
		float64(ServHandOp*Goroutines) / float64(ComputationTime.Load()),
		float64(Latency.Load()),
		float64(AllocationTime.Load()),
		float64(DeallocationTime.Load())}
}
