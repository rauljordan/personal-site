---
layout: post
title:  "Reuse Expensive Computation With In-Progress Caches in Go"
preview: How to leverage channels to prevent duplicating the same, expensive work in Go
date: 2021-Jan-05
tags: 
  - golang
---

Caching is the go-to solution in applications to avoid repeating expensive computation and instead prefer some value that can be readily fetched in-memory. A simple caching strategy is to use a cache as a thin layer above database read access as follows:

```go
package main

import "sync"

type Database struct {
	cache map[string][]byte
	lock  sync.RWMutex
}

func (db *Database) GetItem(key []byte) ([]byte, error) {
	db.lock.RLock()
	if value, ok := db.cache[string(key)]; ok {
		db.lock.RUnlock()
		return value
	}
	db.lock.RUnlock()
	return db.readFromDatabase(key)
}

func (db *Database) WriteItem(key, value []byte) error {
	if err := db.writeToDatabase(key, value); err != nil {
		return err
	}
	db.lock.Lock()
	db.cache[string(key)] = value
	db.lock.Unlock()
	return nil
}
```

This strategy works great for applications where you have requests to read access for a certain value repeatedly, preventing you from performing a potentially expensive db query and leveraging fast access in-memory. Caching is very helpful. For some problems, however, a cache is definitely not enough.

### The busy workers problem

Imagine you have thousands or more processes attempting to perform the same expensive computation at the same time. Perhaps all of them were notified they need to crunch certain numbers which takes a long time, or they need perform a prohibitively expensive operation that can max out your CPU or RAM if overdone. This is quite a common problem in my project, Prysm, which has many different workers in the form of goroutines often attempting to perform duplicate work. A naive solution to this is to simply leverage a cache strategy to avoid repeated computation, as shown above. However, what if there is nothing in the cache yet for the value you care about and thousands of workers are _already_ attempting the expensive computation? Perhaps there are many workers attempting to perform an action that is _already in progress_. This is a great use-case for what we call an **in progress cache**. Let's look at an example:

```go
package main

import "sync"

func main() {
	var wg sync.WaitGroup
	numWorkers := 1000
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func(w *sync.WaitGroup) {
			defer wg.Done()
			doSomethingExpensive()
		}(&wg)
	}
	wg.Wait()
}

func doSomethingExpensive() {
	// Get result from cache if it has already completed.
	value, ok := checkCache()
	ok{
		// Do something with the cached value.
	}

	// Expensive operation which can take a few seconds to complete...
}
```

But what if there is nothing in the cache yet when all 1000 workers are attempting to perform the expensive operation? Well, all of them will start performing your expensive operation, your computer might blow up, and our cache was pretty much useless. Instead, we can leverage the power of **Go channels** to mark work as **in progress** and instead have all workers share the same return value of whichever worker completed it first. Let's think about how to do this.

First of all, we need a way to _block_ a worker from performing expensive computation if a request we care about is already in progress. Second, once a _single worker completes_ an expensive computation, we need to notify all workers that were attempting the same computation of the return value immediately. To accomplish the first task, we can leverage a combination of a shared map to check if a request is in progress, then subscribe to the in-progress request by initializing a channel and appending it to some shared list for the request. Finally, once a worker completes the computation, it can send out the result to all receivers subscribed to that in-progress request. Let's see it in action.

```go
type service struct {
	inProgress         map[string]bool
	awaitingCompletion map[string][]chan string
	lock               sync.RWMutex
}
```

Above, we define a simple struct used to encapsulate this information. In our example, the result of our expensive computation is some string value and the request identity is also a string. We keep track of two maps for request identities: the first is called `inProgress` and will be used by workers to check if expensive computation is already in progress. The second is called `awaitingCompletion`, which is a list of channels that are awaiting to be notified of an in-progress request. They are essentially other workers that are subscribing to the computed value of the worker currently in progress. We use a mutex to make these maps thread-safe.

Next up, we start our `main` function simulating 5 workers doing some expensive operation concurrently.

```go
func main() {
	ss := &service{
		inProgress:         make(map[string]bool),
		awaitingCompletion: make(map[string][]chan string),
	}
	// Create N = 5 workers.
	numWorkers := 5
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	// Launch N goroutines performing the same work:
	// a request with ID "expensivecomputation".
	requestID := "expensivecomputation"
	for i := 0; i < numWorkers; i++ {
		go func(w *sync.WaitGroup, id string) {
			defer wg.Done()
			ss.doWork(id)
		}(&wg, requestID)
	}

	// Wait for all goroutines to complete work.
	wg.Wait()
	fmt.Println("Done")
}
```

Next up, we look at the key function: `doWork(requestID string)`. We'll write it out in Go pseudocode first.

```go
package main

import "time"

func (s *service) doWork(requestID string) {
	if ok := s.inProgress[requestID]; ok {
		// Subscribe to be notified of when the in-progress
		// request completes via a channel.

		// Await the response from the worker currently in-progress...

		return
	}

	// Mark the requestID as in progress.
	s.lock.Lock()
	s.inProgress[requestID] = true
	s.lock.Unlock()

	// Perform some expensive, lengthy work (time.Sleep used to simulate it).
	time.Sleep(time.Second * 4)
	response := "the answer is 42"

	// Send to all subscribers.
	s.lock.RLock()
	receiversWaiting, ok := s.awaitingCompletion[requestID]
	s.lock.RUnlock()
	if ok {
		for _, ch := range receiversWaiting {
			ch <- response
		}
	}

	// Reset the in-progress data for the requestID.
	s.lock.Lock()
	s.inProgress[requestID] = false
	s.awaitingCompletion[requestID] = make([]chan string, 0)
	s.lock.Unlock()
}
```

We lock around the map access to reduce lock contention in the real application. Next up, we fill in the logic for `if ok := inProgress[key]; ok`.

```go
if ok := s.inProgress[requestID]; ok {
  // We add a buffer of 1 to prevent blocking
  // the sender's goroutine.
  responseChan := make(chan string, 1)
  defer close(responseChan)

  lock.Lock()
  s.awaitingCompletion[requestID] = append(s.awaitingCompletion[requestID], responseChan)
  lock.Unlock()
  fmt.Println("Awaiting work in-progress")
  val := <-responseChan
  fmt.Printf("Work result received with value %s\n", val)
  return
}
```

Putting it altogether, we get:

```go
package main

import (
	"fmt"
	"sync"
	"time"
)

type service struct {
	inProgress         map[string]bool
	awaitingCompletion map[string][]chan string
	lock               sync.RWMutex
}

func (s *service) doWork(requestID string) {
	s.lock.RLock()
	if ok := s.inProgress[requestID]; ok {
		s.lock.RUnlock()
		responseChan := make(chan string, 1)
		defer close(responseChan)

		s.lock.Lock()
		s.awaitingCompletion[requestID] = append(s.awaitingCompletion[requestID], responseChan)
		s.lock.Unlock()
		fmt.Println("Awaiting work completed")
		val := <-responseChan
		fmt.Printf("Work result received with value %s\n", val)
		return
	}
	s.lock.RUnlock()

	s.lock.Lock()
	s.inProgress[requestID] = true
	s.lock.Unlock()

	// Do expensive operation
	fmt.Println("Doing expensive work...")
	time.Sleep(time.Second * 4)
	result := "the answer is 42"

	s.lock.RLock()
	receiversWaiting, ok := s.awaitingCompletion[requestID]
	s.lock.RUnlock()

	if ok {
		for _, ch := range receiversWaiting {
			ch <- result
		}
		fmt.Println("Sent result to all subscribers")
	}

	s.lock.Lock()
	s.inProgress[requestID] = false
	s.awaitingCompletion[requestID] = make([]chan string, 0)
	s.lock.Unlock()
}

func main() {
	ss := &service{
		inProgress:         make(map[string]bool),
		awaitingCompletion: make(map[string][]chan string),
	}
	numWorkers := 5
	var wg sync.WaitGroup
	wg.Add(numWorkers)
	requestID := "expensivecomputation"
	for i := 0; i < numWorkers; i++ {
		go func(w *sync.WaitGroup, id string) {
			defer wg.Done()
			ss.doWork(id)
		}(&wg, requestID)
	}
	wg.Wait()
	fmt.Println("Done")
}
```

Now running the main.go file: `go run main.go`, we observe it happening as expected:

```text
Doing expensive work...
Awaiting work completed
Awaiting work completed
Awaiting work completed
Awaiting work completed
Sent result to all subscribers
Work result received with value the answer is 42
Work result received with value the answer is 42
Work result received with value the answer is 42
Work result received with value the answer is 42
Done
```

One out of 5 workers is doing the expensive work, the rest are waiting for it to complete. Once it completes after 4 seconds, the 4 subscribed workers receive the value correctly "the answer is 42"! Hopefully this simple approach can help you when you want to reduce duplicate work performed by your background routines, leveraging the power of Go channels to block goroutines until a value is received.

**NOTE**: The code above is not meant for production, as in production you need to have better ways of dealing with goroutine context cancelation and a smarter way of namespacing requests and subscribers rather than just using naive maps.
