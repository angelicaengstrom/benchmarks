package benchmarks

import (
	"fmt"
	"math/rand/v2"
)

func producing(buffer chan int, done chan bool, op int) {
	for i := 0; i < op; i++ {
		x := rand.IntN(100)
		buffer <- x
	}
	done <- true
}

func consuming(buffer chan int, done chan bool) {
	for x := range buffer {
		fmt.Print(x, " ")
	}
	done <- true
}

func RunProducerConsumer(n int, op int) {
	buffer := make(chan int)
	doneProducers := make(chan bool)
	doneConsumers := make(chan bool)

	for i := 0; i < n; i++ {
		go producing(buffer, doneProducers, op)
		go consuming(buffer, doneConsumers)
	}

	go func() {
		for i := 0; i < n; i++ {
			<-doneProducers
		}
		close(buffer)
	}()

	for i := 0; i < n; i++ {
		<-doneConsumers
	}
}
