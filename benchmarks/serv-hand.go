package benchmarks

import (
	"fmt"
	"net"
	"time"
)

type server struct {
	connection chan net.Conn
	listener   net.Listener
}

func newServer(address string) (*server, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("error listening on %s: %w", address, err)
	}
	return &server{listener: listener, connection: make(chan net.Conn)}, nil
}

func (s *server) acceptConnections() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			continue
		}
		select {
		case s.connection <- conn:
		default:
			conn.Close()
			return
		}
	}
}

func (s *server) handleConnections() {
	for {
		select {
		case conn := <-s.connection:
			go s.handleConnection(conn)
		default:
			return
		}
	}
}

func (s *server) handleConnection(conn net.Conn) {
	defer conn.Close()
	fmt.Fprintf(conn, "Hello, world!")
	time.Sleep(1 * time.Second)
	fmt.Fprintf(conn, "Goodbye, world!")
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
	for i := 0; i < op; i++ {
		_, err := net.Dial("tcp", address)
		if err != nil {
			fmt.Printf("error connecting to %s: %s\n", address, err)
		}
	}
	done <- true
}

func RunServerHandler(n int, op int) {
	address := ":8080"
	s, err := newServer(address)
	if err != nil {
		fmt.Println(err)
		return
	}
	done := make(chan bool)
	s.run()

	for i := 0; i < n; i++ {
		go sendRequests(op, done, address)
	}

	for i := 0; i < n; i++ {
		<-done
	}
	if s.stop() != nil {
		fmt.Println("Could not stop server")
	}

}
