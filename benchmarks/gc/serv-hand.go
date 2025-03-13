package gc

import (
	. "experiments/benchmarks/metrics"
	"fmt"
	"net"
	"runtime"
	"runtime/debug"
	"time"
)

type server struct {
	connection chan net.Conn
	listener   net.Listener
}

func newServer(address string) (*server, error) {
	allocationTimeStart := time.Now()
	listener := new(net.Listener)
	s := new(server)
	s.connection = make(chan net.Conn)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	*listener, _ = net.Listen("tcp", address)
	s.listener = *listener

	return s, nil
}

func (s *server) acceptConnections() {
	for {
		allocationTimeStart := time.Now()
		conn := new(net.Conn)
		err := new(error)
		AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

		*conn, *err = s.listener.Accept()
		if *err != nil {
			continue
		}
		select {
		case s.connection <- *conn:
		default:
			(*conn).Close()
			return
		}
	}
}

func (s *server) handleConnections() {
	allocationTimeStart := time.Now()
	conn := new(net.Conn)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	var memStats runtime.MemStats
	latencyStart := time.Now()
	for *conn = range s.connection {
		Latency.Add(time.Since(latencyStart).Nanoseconds())
		s.handleConnection(*conn)
		latencyStart = time.Now()

		runtime.ReadMemStats(&memStats)

		if memStats.HeapAlloc > P_memoryConsuption.Load() {
			P_memoryConsuption.Store(memStats.HeapAlloc)
		}

		externalFrag := float64(memStats.HeapIdle) / float64(memStats.HeapSys)
		if externalFrag > P_externalFrag.Load().(float64) {
			P_externalFrag.Store(externalFrag)
		}

		internalFrag := float64(memStats.HeapIntFrag) / float64(memStats.HeapAlloc)
		if internalFrag > P_internalFrag.Load().(float64) {
			P_internalFrag.Store(internalFrag)
		}
	}
}

func (s *server) handleConnection(conn net.Conn) {
	_ = conn
	conn.Close()
}

func (s *server) run() {
	go s.acceptConnections()
	go s.handleConnections()
}

func (s *server) stop() error {
	close(s.connection)
	return s.listener.Close()
}

func sendRequests(op int, done chan bool, address string) {
	for i := new(int); *i < op; *i++ {
		_, _ = net.Dial("tcp", address)
	}
	done <- true
}

func RunServerHandler() Metrics {
	debug.SetGCPercent(-1)

	ComputationTime.Store(0)
	AllocationTime.Store(0)
	DeallocationTime.Store(0)
	Latency.Store(0)
	P_memoryConsuption.Store(0)
	P_internalFrag.Store(0.0)
	P_externalFrag.Store(0.0)

	computationTimeStart := time.Now()

	allocationTimeStart := time.Now()
	address := new(string)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	*address = ":8080"

	s, err := newServer(*address)
	if err != nil {
		fmt.Println(err)
		return Metrics{}
	}

	allocationTimeStart = time.Now()
	done := make(chan bool)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	s.run()

	for i := 0; i < Goroutines; i++ {
		go sendRequests(ServHandOp, done, *address)
	}

	for i := 0; i < Goroutines; i++ {
		<-done
	}

	if s.stop() != nil {
		fmt.Println("Could not stop server")
	}

	ComputationTime.Store(time.Since(computationTimeStart).Nanoseconds())

	deallocationStart := time.Now()
	runtime.GC()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	return Metrics{
		float64(ComputationTime.Load()) / 1_000_000_000,
		float64(ServHandOp*1_000_000_000) / float64(ComputationTime.Load()),
		float64(Latency.Load()) / 1_000_000_000,
		float64(P_memoryConsuption.Load()),
		P_externalFrag.Load().(float64),
		P_internalFrag.Load().(float64),
		float64(AllocationTime.Load()) / 1_000_000_000,
		float64(DeallocationTime.Load()) / 1_000_000_000}
}
