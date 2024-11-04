package pool_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
	"vk/pool"
)

func captureOutput(f func()) string {
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestNewWorkerPool(t *testing.T) {
	jobs := make(chan string)
	output := captureOutput(func() {
		pool := pool.New(3, jobs)
		defer pool.Close()
	})

	println(output)
	addedCount := strings.Count(output, "added")
	expected := 3
	if addedCount != expected {
		t.Errorf("Ожидалось добавить %d воркеров, но добавлено %d", expected, addedCount)
	}
}

func TestAddWorker(t *testing.T) {
	jobs := make(chan string)
	poolInstance := pool.New(1, jobs)
	defer poolInstance.Close()

	output := captureOutput(func() {
		poolInstance.AddWorker()
		time.Sleep(100 * time.Millisecond)
	})

	if !strings.Contains(output, "Worker 2 added.") {
		t.Errorf("Ожидалось сообщение о добавлении второго воркера, но не найдено")
	}
}

func TestRemoveWorker(t *testing.T) {
	jobs := make(chan string)
	poolInstance := pool.New(2, jobs)
	defer poolInstance.Close()

	output := captureOutput(func() {
		poolInstance.RemoveWorker()
		time.Sleep(100 * time.Millisecond)
	})

	if !strings.Contains(output, "Worker 2 stopped.") {
		t.Errorf("Ожидалось сообщение о остановке второго воркера, но не найдено")
	}
}

func TestRemoveWorkerWhenNone(t *testing.T) {
	jobs := make(chan string)
	poolInstance := pool.New(0, jobs)
	defer poolInstance.Close()

	output := captureOutput(func() {
		poolInstance.RemoveWorker()
	})

	if !strings.Contains(output, "No workers to remove.") {
		t.Errorf("Ожидалось сообщение о отсутствии воркеров для удаления, но не найдено")
	}
}

func TestProcessJobs(t *testing.T) {
	jobs := make(chan string)
	poolInstance := pool.New(2, jobs)
	defer poolInstance.Close()

	jobList := []string{"job1", "job2", "job3", "job4"}

	output := captureOutput(func() {
		for _, job := range jobList {
			jobs <- job
		}
		time.Sleep(2 * time.Second)
	})

	println(output)

	for _, job := range jobList {
		if !strings.Contains(output, fmt.Sprintf("processing job: %s", job)) {
			t.Errorf("Ожидалось обработать задание %s, но оно не было найдено в выводе", job)
		}
	}
}

func TestClosePool(t *testing.T) {
	jobs := make(chan string)
	poolInstance := pool.New(2, jobs)

	output := captureOutput(func() {
		poolInstance.Close()
	})

	if !strings.Contains(output, "Worker 1 stopped.") || !strings.Contains(output, "Worker 2 stopped.") {
		t.Errorf("Ожидалось сообщение об остановке всех воркеров, но не все сообщения найдены")
	}
}

func TestConcurrentAddRemove(t *testing.T) {
	jobs := make(chan string)
	poolInstance := pool.New(2, jobs)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			poolInstance.AddWorker()
			time.Sleep(10 * time.Millisecond)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			poolInstance.RemoveWorker()
			time.Sleep(15 * time.Millisecond)
		}
	}()

	wg.Wait()

	output := captureOutput(func() {
		poolInstance.Close()
	})

	stopCount := strings.Count(output, "Worker")
	if stopCount < 2 {
		t.Errorf("Ожидалось как минимум остановку 2 воркеров, но остановлено %d", stopCount)
	}
}

func TestNoDeadlock(t *testing.T) {
	jobs := make(chan string)
	poolInstance := pool.New(3, jobs)

	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			poolInstance.AddWorker()
			poolInstance.RemoveWorker()
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Errorf("Тест зациклился при добавлении и удалении воркеров")
	}

	poolInstance.Close()
}

func TestAddRemoveWorkersInParallel(t *testing.T) {
	jobs := make(chan string)
	poolInstance := pool.New(5, jobs)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			poolInstance.AddWorker()
			time.Sleep(5 * time.Millisecond)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			poolInstance.RemoveWorker()
			time.Sleep(7 * time.Millisecond)
		}
	}()

	wg.Wait()

	output := captureOutput(func() {
		poolInstance.Close()
	})

	stopCount := strings.Count(output, "Worker")
	if stopCount < 5 {
		t.Errorf("Ожидалось остановить как минимум 5 воркеров, но остановлено %d", stopCount)
	}
}

func TestAddRemoveAfterClose(t *testing.T) {
	jobs := make(chan string)
	poolInstance := pool.New(2, jobs)

	outputAdd := captureOutput(func() {
		poolInstance.Close()
		poolInstance.AddWorker()
	})

	outputRemove := captureOutput(func() {
		poolInstance.RemoveWorker()
	})

	if !strings.Contains(outputAdd, "Pool is closed.") {
		t.Errorf("Не ожидалось добавление воркера после закрытия пула")
	}

	if !strings.Contains(outputRemove, "Pool is closed.") {
		t.Errorf("Ожидалось сообщение о отсутствии воркеров для удаления после закрытия пула")
	}
}
