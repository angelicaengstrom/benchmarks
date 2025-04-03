package gc

import (
	. "experiments/benchmarks/metrics"
	"fmt"
	"net"
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

func newServer(address string) (*server, error) {
	allocationTimeStart := time.Now()
	s := new(server)
	s.requests = make(chan Request)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	s.listener, _ = net.Listen("tcp", address)

	return s, nil
}

func (s *server) acceptConnections(done chan bool, req *Request) {
	var i *int
	for i = new(int); *i < ServHandOp*Goroutines; *i++ {
		allocationTimeStart := time.Now()
		req = new(Request)
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
	done <- true
}

func (s *server) handleConnections(done chan bool, req *Request) {
	allocationTimeStart := time.Now()
	req = new(Request)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	for *req = range s.requests {
		Latency.Add(time.Since(req.latencyStart).Nanoseconds())
		s.handleConnection(req)
	}
	done <- true
}

func (s *server) handleConnection(req *Request) {
	req.conn.Close()
}

var accReq Request
var handReq Request

func (s *server) run(done chan bool) {
	go s.acceptConnections(done, &accReq)
	go s.handleConnections(done, &handReq)
}

func (s *server) stop(done chan bool) error {
	<-done // acceptConnections
	<-done // handleConnections
	return s.listener.Close()
}

func sendRequests(op int, done chan bool, address string, i *int, req *Request) {
	for i = new(int); *i < op; *i++ {
		allocationTimeStart := time.Now()
		req = new(Request)
		AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

		req.conn, _ = net.Dial("tcp", address)

		req.conn.Close()
	}
	done <- true
}

func RunServerHandler() SystemMetrics {
	debug.SetGCPercent(-1)

	ComputationTime.Store(0)
	AllocationTime.Store(0)
	DeallocationTime.Store(0)
	Latency.Store(0)

	// Bypassing escaping
	c := [Goroutines]int{}
	conn := [Goroutines]Request{}

	computationTimeStart := time.Now()

	allocationTimeStart := time.Now()
	address := new(string)
	done := make(chan bool)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())
	*address = ":8080"

	s, err := newServer(*address)
	if err != nil {
		fmt.Println(err)
		return SystemMetrics{}
	}

	s.run(done)

	for i := 0; i < Goroutines; i++ {
		go sendRequests(ServHandOp, done, *address, &c[i], &conn[i])
	}

	for i := 0; i < Goroutines; i++ {
		<-done
	}

	if s.stop(done) != nil {
		fmt.Println("Could not stop server")
	}

	deallocationStart := time.Now()
	runtime.GC()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	ComputationTime.Store(time.Since(computationTimeStart).Nanoseconds())

	return SystemMetrics{
		float64(ComputationTime.Load()),
		float64(ServHandOp*Goroutines) / float64(ComputationTime.Load()),
		float64(Latency.Load()),
		float64(AllocationTime.Load()),
		float64(DeallocationTime.Load())}
}
