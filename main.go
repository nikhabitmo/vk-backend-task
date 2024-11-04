package main

import (
	"fmt"
	"time"
	"vk/pool"
)

func main() {
	jobs := make(chan string, 100)

	p := pool.New(10, jobs)

	for i := 1; i <= 100; i++ {
		if i%20 == 0 {
			p.RemoveWorker()
		}

		jobs <- fmt.Sprintf("Job %d", i)
	}

	for i := 1; i <= 5; i++ {
		p.RemoveWorker()
	}

	time.Sleep(2 * time.Second)
	p.RemoveWorker()

	time.Sleep(2 * time.Second)
	p.RemoveWorker()

	p.Close()
}
