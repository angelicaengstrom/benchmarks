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

type Conn struct {
	c            net.Conn
	latencyStart time.Time
}

type server struct {
	connection chan Conn
	listener   net.Listener
}

func newServer(address string, r *region.Region) (*server, error) {
	allocationTimeStart := time.Now()
	listener := region.AllocFromRegion[net.Listener](r)
	s := region.AllocFromRegion[server](r)
	s.connection = region.AllocChannel[Conn](0, r)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	*listener, _ = net.Listen("tcp", address)
	s.listener = *listener

	return s, nil
}

func (s *server) acceptConnections(r *region.Region) {
	for {
		allocationTimeStart := time.Now()
		conn := region.AllocFromRegion[net.Conn](r)
		err := region.AllocFromRegion[error](r)
		AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

		*conn, *err = s.listener.Accept()
		if *err != nil {
			continue
		}
		select {
		case s.connection <- Conn{*conn, time.Now()}:
		default:
			(*conn).Close()
			r.DecRefCounter()
			return
		}
	}
}

func (s *server) handleConnections(r *region.Region) {
	allocationTimeStart := time.Now()
	conn := region.AllocFromRegion[Conn](r)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	for *conn = range s.connection {
		Latency.Add(time.Since(conn.latencyStart).Nanoseconds())
		r.IncRefCounter()
		s.handleConnection(conn.c, r)
	}
	r.DecRefCounter()
}

func (s *server) handleConnection(conn net.Conn, r *region.Region) {
	_ = conn
	conn.Close()
	r.DecRefCounter()
}

func (s *server) run(r *region.Region) {
	r.IncRefCounter()
	go s.acceptConnections(r)
	r.IncRefCounter()
	go s.handleConnections(r)
}

func (s *server) stop() error {
	close(s.connection)
	return s.listener.Close()
}

func sendRequests(op int, done chan bool, address string, r *region.Region) {
	for i := region.AllocFromRegion[int](r); *i < op; *i++ {
		_, _ = net.Dial("tcp", address)
	}
	done <- true
	r.DecRefCounter()
}

func RunServerHandler() Metrics {
	debug.SetGCPercent(-1)

	ComputationTime.Store(0)
	AllocationTime.Store(0)
	DeallocationTime.Store(0)
	Latency.Store(0)

	computationTimeStart := time.Now()
	r1 := region.CreateRegion()

	allocationTimeStart := time.Now()
	address := region.AllocFromRegion[string](r1)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	*address = ":8080"

	s, err := newServer(*address, r1)
	if err != nil {
		fmt.Println(err)
		return Metrics{}
	}

	allocationTimeStart = time.Now()
	done := region.AllocChannel[bool](0, r1)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	s.run(r1)

	for i := 0; i < Goroutines; i++ {
		r1.IncRefCounter()
		go sendRequests(ServHandOp, done, *address, r1)
	}

	for i := 0; i < Goroutines; i++ {
		<-done
	}

	if s.stop() != nil {
		fmt.Println("Could not stop server")
	}

	ComputationTime.Store(time.Since(computationTimeStart).Nanoseconds())

	deallocationStart := time.Now()
	r1.RemoveRegion()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	runtime.GC()

	return Metrics{
		float64(ComputationTime.Load()) / 1_000,
		float64(ServHandOp*1_000_000) / float64(ComputationTime.Load()),
		float64(Latency.Load()) / 1_000,
		float64(AllocationTime.Load()) / 1_000,
		float64(DeallocationTime.Load()) / 1_000}
}
