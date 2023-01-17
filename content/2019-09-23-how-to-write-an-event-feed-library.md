+++
title =  "Writing an One-to-Many Event Feed Library in Go"
date = 2019-09-23

[taxonomies]
tags = ["golang"]

[extra]
photo = ""
+++

```go
type sub struct {
    feed         *BoolFeed
    channelIndex int
    channel      chan bool
    once         sync.Once
    err          chan error
}
```

This blog post explores the design rationale behind building a performant, simple, one-to-many event feed library in Go. We'll be recreating the event library from the [go-ethereum](https://github.com/ethereum/go-ethereum/blob/master/event/feed.go) project step by step, even explaining some of the tricky design decisions behind its robust concurrency approach. 

<!-- more -->

# Ping Pong

When I first started learning Go, I didn't truly understand the _fantastic_ concurrency primitives the language provides. Most real world applications tend to be concurrent and asynchronous...meaning all sorts of things can and will happen simultaneously and at different time intervals, often beyond our control. A big challenge when building concurrent applications is then the control and flow of shared state information, often becoming a very complex problem. In Go, communication between concurrent threads happens through special data structures called **channels**, which are _typed_ and allow for a basic "sender" vs. "listener" design pattern. This pattern, as we'll soon see, is more expressive than it seems, giving Go one of the strongest toolkits to handle even the most complex concurrency challenges.

Let's take a look at a basic example:

```go
package main

import (
    "fmt"
    "time"
)

func main() {
    ch := make(chan string)
    go ping(ch)
    go pong(ch)
    select {}
}

func ping(ch chan string) {
    for {
        time.Sleep(time.Second)
        fmt.Println("Sending message...")
        ch <- "ping"
    }
}

func pong(ch chan string) {
    for {
        select {
        case p :=<-ch:
            fmt.Printf("Received message %s\n", p)
            if p == "ping" {
                fmt.Println("pong")
            }
        }
    }
}
```

In the example above, we created two functions, `ping` and `pong`. The first one sends the "ping" message over a string typed channel, while the other listens for and receives the message, checking its value, and logging "pong" accordingly. We spawn off both functions concurrently using the `go` keyword, which creates a lightweight thread called a _goroutine_ that runs alongside the main thread of the program. Having channels abstracts away the complexity of goroutine communication, making users unaware of the concurrency details happening underneath the hood and leaving us to simply focus on our core application logic.

![image](https://toivjon.files.wordpress.com/2017/11/javafx-pong.png?w=772)

## Gotchas: Buffered vs. Unbuffered Channels

When creating a Go channel, we have the option of making it either buffered or unbuffered. An unbuffered channel is initialized with a simple make call along with its type, `make(chan bool)`. A buffered channel instead specifies a `buffer` length, such as `make(chan bool, 5)`. Think of channels as the mailman, bringing packages all across the neighborhood. The buffer is the the capacity of his mailbag, containing all the envelopes. The mailman will wait till his bag is full before making a trip to deliver the letters. Whenever you specify a buffered channel, that's what happens. Essentially, data is kept in the buffer until it fills up to capacity, and at that point a listener or receiver can read all of the information contains in the channel. An unbuffered channel, however, has a capacity of 0, so it can't store data. Every send over an unbuffered channel _must_ have a recipient ready to process the send right away.

Let's take a look at a basic unbuffered example:

```go
func main() {
    ch := make(chan string)
    go func() {
        ch <- "hello world"
    }()
    msg := <-ch
    fmt.Println(msg)
}
```

We create a listener over our basic, unbuffered channel and log once we receive it, great! If we run this program we see our little "hello world" pop up on our terminal window. If we never send, the program will wait forever, given the recipient will hang there never receiving a message to proceed into the Println statement.

```go
func main() {
    ch := make(chan string, 1)
    go func() {
        ch <- "hello world"
    }()
    msg := <-ch
    fmt.Println(msg)
}
```

What happens if we try to send and receive in the same goroutine with an unbuffered channel?

```go
func main() {
    ch := make(chan string)
    ch <- "hello world"
    msg := <-ch
    fmt.Println(msg)
}
```

Running...

```bash
$ go run main.go
fatal error: all goroutines are asleep - deadlock!

goroutine 1 [chan send]:
main.main()
```

![image](https://raw.githubusercontent.com/MariaLetta/free-gophers-pack/master/illustrations/png/2.png)

Yikes! Well the issue is pretty clear here, as there's no listener, the channel will _block_ the main thread on sending, preventing it from advancing at all! Even if we add a listener right beneath the send, we will get the same error because the code can _never_ proceed past `ch <- "hello world"`, as the send call never even completes. There must already be a listener prepared to receive when the `send` is called. 

> You cannot send and receive from an unbuffered channel in the same goroutine, as there needs to be a receiver ready before you send anything out!

The code above works if we initialize our channel with a buffer `ch := make(chan string, 1)`, as the channel will then store the data until a listener reads it even if there is no listener ready at the time of sending.

## Why Buffered Channels Are Great

Buffered channels have a fundamentally different use case than unbuffered ones, as by their very nature, they're well-suited to handle aggregation of information across goroutines. For example, say you're creating a concurrent web scraper that tries to scrape a ton of websites at the same time, and eventually you want to aggregate the results into some nice form for final processing. If you know the list of websites you'll be scraping ahead of time, you can create a buffered channel of that length, do the heavy lifting across a bunch of concurrent goroutines, and write the results to this channel. When you're done, a listener will be ready to loop over the aggregated results and handle them accordingly. Let's take a look:

```go
type ScrapedData struct {
    Url      string
    Html     string
    NumLinks int
}

func scrapeSite(s string) *ScrapedData {
    // Do some intense data scraping...
    htmlResult := getHtmlResults(s)
    numLinks := getNumLinks(htmlResult)
    ...
    return &ScrapedData{
        Url: s,
        Html:     htmlResults,
        NumLinks: numLinks,
    }
}

func main() {
    websites := []string{"example.com", "wikipedia.org"}
    ch := make(chan *ScrapedData, len(websites))

    for i := 0; i < len(websites); i++ {
        go func(ii int) {
            ch <- scrapeSite(websites[ii])
        }(i)
    }

    for val := range ch {
        fmt.Printf("Scraped %s with number of links %d\n", val.Url, val.NumLinks)
    }
}
```

Using the concepts from before, we declare a _buffered_ channel of type `*ScrapedData`. Then, for every website, we spawn off a goroutine that does some heavy data scraping and writes the result to the channel. In the end, we loop over the results and print out every single value we just scraped. Does the code above run? Let's see:

```bash
$ go run main.go
Scraped example.com with number of links 10
Scraped wikipedia.org with number of links 20
fatal error: all goroutines are asleep - deadlock!

goroutine 1 [chan receive]:
main.main()
```

Shoot - what the heck happened here? As it turns out, Go's `range` function isn't smart enough to figure out we're done writing into the channel, and that's not its fault! Given we're building some concurrent application, there could be some other routine that tries to keep writing into the channel, so there's no way for `range` to know we're done writing, even if the channel is filled up to the brim. Instead, once we're done writing and we're sure of it, we should call Go's built-in `close` operator to close the channel off for future communication.

```go
func main() {
    websites := []string{"example.com", "wikipedia.org"}
    ch := make(chan *ScrapedData, len(websites))

    for i := 0; i < len(websites); i++ {
        go func(ii int) {
            ch <- scrapeSite(websites[ii])
        }(i)
    }

    close(ch)

    for val := range ch {
        fmt.Printf("Scraped %s with number of links %d\n", val.Url, val.NumLinks)
    }
}
```

Turns out this eliminates the deadlock error, but nothing else gets printed out...what gives? Concurrent applications are often _asynchronous_, meaning we don't know when they'll complete relative to other threads and we should not make any assumptions regarding this. In the code above, when we spawn off two goroutines, the main thread keeps going, there's nothing stopping it! _So the `close` call might have happened way before we attempt to scrape the data_, leaving us in a weird spot. As a solution, Go provides an awesome tool called a `WaitGroup`, which lets us block threads until a specified number of goroutines complete.

```go
func main() {
    websites := []string{"example.com", "wikipedia.org"}
    ch := make(chan *ScrapedData, len(websites))

    var wg sync.WaitGroup
    wg.Add(len(websites))
    for i := 0; i < len(websites); i++ {
        go func(ii int, w *sync.WaitGroup) {
            ch <- scrapeSite(websites[ii])
            w.Done()
        }(i, &wg)
    }

    wg.Wait()
    close(ch)

    for val := range ch {
        fmt.Printf("Scraped %s with number of links %d\n", val.Url, val.NumLinks)
    }
}
```

Using a wait group, we can tell the program "hey, we have 2 goroutines that are scheduled, wait until they're finished before proceeding". When the goroutine finishes, we can call `wg.Done()` to notify the wait group. Note we pass in a reference to the wait group in the goroutine's arguments to ensure we don't make a different copy of the wait group we initialized. _Now_, when we run the code again, we'll wait till all routines complete before we close off the channel, and we should have a fully functional program now with wait groups, fancy buffered channels, and even a range loop to read the aggregated data results :).

```bash
$ go run main.go
Scraped example.com with number of links 10
Scraped wikipedia.org with number of links 20
```

## One-to-Many Subscriptions

Now that we reviewed buffered and unbuffered channels as well as goroutines, let's see what else this powerful construct allows us to build. So far, we've only seen single sender vs. receiver patterns, where there's a single listener waiting for results sent into some channel. But what if we want many listeners to receive the same data and do something with it _simultaneously_? This should work, right?

```go
var userID = 0

type UserInfo struct {
    ID int
}

func main() {
    ch := make(chan *UserInfo)
    go simulateUserSignup(ch)
    go sendEmail(ch)
    go saveUserToDB(ch)
    select {}
}

func simulateUserSignup(ch chan *UserInfo) {
    for {
        time.Sleep(time.Second)
        ch <- &UserInfo{
            ID: userID,
        }
        userID++
    }
}

func sendEmail(ch chan *UserInfo) {
    for {
        select {
        case user := <-ch:
            fmt.Printf("Sending confirmation email to user with ID %d...\n", user.ID)
        }
    }
}

func saveUserToDB(ch chan *UserInfo) {
    for {
        select {
        case user := <-ch:
            fmt.Printf("Saving user data to database with ID %d...\n", user.ID)
        }
    }
}
```

The output...

```bash
$ go run main.go
Saving user data to database with ID 0...
Sending confirmation email to user with ID 1...
Saving user data to database with ID 2...
Sending confirmation email to user with ID 3...
Saving user data to database with ID 4...
Sending confirmation email to user with ID 5...
Saving user data to database with ID 6...
Sending confirmation email to user with ID 7...
Saving user data to database with ID 8...
```

Oh no...we _really_ don't that to happen. We want **both** the sending of a confirmation email + the saving a user to the DB to happen together, not skipping every other user! Turns out, naïve channels don't exactly work when you need a one-to-many subscription pattern. We could indeed rearchitect the code above to instead listen once and perform both actions sequentially and synchronously:

```go
func main() {
    ch := make(chan *UserInfo)
    go simulateUserSignup(ch)
    for {
        select {
        case user := <-ch:
            fmt.Printf("Saving user data to database with ID %d...\n", user.ID)
            // Perform the db save...
            fmt.Printf("Sending confirmation email to user with ID %d...\n", user.ID)
            // Send the confirmation email...
        }
    }
}
```

And sure, this is the _best_ solution if your entire application is a single event loop, but more complex applications typically have various services running concurrently with their own event loops, and having everything work in this synchronous pattern isn't always the best. Sometimes, we want to simply trigger events in our applications without worrying too much about who's listening, with the possibility of having infinitely many listeners that can conditionally **subscribe** to the event notification. This common pattern is conventionally referred to as `PubSub`. A naïve implementation of a one-to-many pubsub with a single channel won't work for this use case, as we see above, there can only be at most 1 listener that receives a single send at any given time. However, can we use these channel primitives to come with a one-to-many implementation that is simple and robust enough to use in production?

## Designing a Subscription and Event Feed Model

We're going to be designing a `Feed` data structure that allows many listeners to subscribe to a single event sent over the feed and receive them as they happen. First, let's outline some feature requirements we'd like to see in any library that implements this pubsub model using Go:

- We want a _generic_ library, I want to be able to trigger events with any sort of data, regardless of its type
- We want the ability for many subscribers to listen for event triggers of a specific type, each receiving the sent data _simultaneously_ as it occurs
- We want the ability for subscribers to _unsubscribe_ whenever they need to, preventing future events being received by such listener
- We want a library that is easy to use, idiomatic, and uses Go's concurrency primitives effectively
- We want a library that is light on memory, performance, and is thread-safe

## A First Pass Implementation

So, where do we start? A good design typically begins by identifying the key invariants of necessary features. We want to focus first on the core desired functionality, which is to allow for **one-to-many** subscriptions of triggered events. To keep things simple, our first implementation will only allow for `bool` type event subscriptions, so let's call our feed a `BoolFeed`.

Given a single channel send can only receive data by at most one listener, why don't we allow our library to send to a bunch of registered channels simultaneously? That is, we can allow listeners to _register_ a channel they want to be notified through, and as we trigger a new event, we send data over all the registered channels via a for loop. Let's give that a shot:

```go
type BoolFeed struct {
    lock sync.Mutex
    listeners []chan bool
}

func (f *BoolFeed) Send(data bool) {
    f.lock.Lock()
    for _, lis := range f.listeners {
        lis<-data
    }
    f.lock.Unlock()
}

func (f *BoolFeed) RegisterListener(lis chan bool) {
    f.lock.Lock()
    f.listeners = append(f.listeners, lis)
    f.lock.Unlock()
}

func firstListener(f *BoolFeed) {
    ch := make(chan bool)
    f.RegisterListener(ch)
    for {
        select {
        case <-ch:
            fmt.Println("Received data in first listener")
        }
    }
}

func secondListener(f *BoolFeed) {
    ch := make(chan bool)
    f.RegisterListener(ch)
    for {
        select {
        case <-ch:
            fmt.Println("Received data in second listener")
        }
    }
}

func main() {
    feed := &BoolFeed{
        listeners: make([]chan bool, 0),
    }

    go func(ff *BoolFeed) {
        for {
            time.Sleep(time.Second)
            ff.Send(true)
        }
    }(feed)

    go firstListener(feed)
    go secondListener(feed)
    select {}
}
```

We use a mutex above to make the operations thread safe, preventing calls to `RegisterListener` from happening at the exact same time as we're doing a send of the data over the currently registered channels. We run the code, and...

```go
$ go run main.go
Received data in second listener
Received data in first listener
Received data in second listener
Received data in first listener
Received data in first listener
```

It works, we see data being properly received by the different listeners. Even though we get the result we expected in this contrived example, _**our naïve approach still has a critical limitation that makes it unsuitable for production**_ which we'll analyze in the next sections.

For now, let's see if we can meet another invariant in our features requirement, which is to unsubscribe from the event triggers whenever want to. For this, we can define a `Subscription` type, which contains an `Unsubscribe()` method, and which can be returned by the `RegisterListener` function. As a subscriber, we don't care about how this unsubscribing by itself works underneath the hood, we just need to have it as an option. For easier testing and implementation, we can define subscriptions as a Go interface:

```go
type Subscription interface {
    Unsubscribe()
}
```

and amend our `BoolFeed` as follows:

```go
func (f *BoolFeed) RegisterListener() Subscription
```

While we're at it, let's make our implementation more idiomatic and rename `RegisterListener` to `Subscribe` to end up with:

```go
func (f *BoolFeed) Subscribe(ch chan bool) Subscription {
    ...
}

func someListener(feed *BoolFeed) {
    ch := make(chan bool)
    sub := feed.Subscribe(ch)
    defer sub.Unsubscribe()
    for {
        select {
        case <-ch:
            fmt.Println("Received data in first listener")
        }
    }
}
```

But what if the feed we're subscribed to all of a sudden has problems, or maybe it belongs to another service that shuts down, how would we know? Indeed, every subscription should have an `error` channel that we can also listen for and unsubscribe if needed.

```go
type Subscription interface {
    Unsubscribe()
    Err() <-chan error
}

func someListener(feed *BoolFeed) {
    ch := make(chan bool)
    sub := feed.Subscribe(ch)
    defer sub.Unsubscribe()
    for {
        select {
        case <-ch:
            fmt.Println("Received data in first listener")
        case err:=<-sub.Err():
            fmt.Printf("Oh no - something went wrong: %v", err)
            return
        }
    }
}
```

Now we have an easy, standard pattern to safely subscribe to and unsubscribe from events should errors occur as we listen for data. As we implement our `Unsubscribe()` function, we should ensure it can only occur **once** to prevent accidental duplicate calls to have detrimental effects.

```go
...

func (f *BoolFeed) Subscribe(lis chan bool) Subscription {
    f.lock.Lock()
    defer f.lock.Unlock()
    f.listeners = append(f.listeners, lis)
    return &sub{
        feed: f,
        channelIndex: len(f.listeners)-1,
        channel: lis,
        err: make(chan error, 1),
    }
}

// removes an channel at index i efficiently as order does not
// matter for listeners we keep track of.
func (f *BoolFeed) remove(i int) {
    f.lock.Lock()
    f.listeners[i] = f.listeners[len(f.listeners)-1]
    f.listeners = f.listeners[:len(f.listeners)-1]
    f.lock.Unlock()
}

type sub struct {
    feed    *BoolFeed
    channelIndex int
    channel chan bool
    once sync.Once
    err     chan error
}

func (s *sub) Unsubscribe() {
    s.once.Do(func() {
        s.feed.remove(s.channelIndex)
        close(sub.err)
    })
}

func (s *sub) Err() <-chan error {
    return s.sub
}
```

Let's digest what's going on above. First, we returned a `&sub{...}` from our `Subscribe` func, storing some information about the channel's index in the listeners, and we also initialize a useful `err` channel as a _buffered channel_. Remember how buffered channels don't block a thread on a send? Well, imagine that the error listener all of a sudden disappears for whatever reason, then if we were writing over an unbuffered channel, we'd have a send call that can never complete, causing us to have memory leakage! This is dangerous thing, as it leads to some of the most silent but deadly bugs you can have in an application. Having a buffered channel of size one ensures that a send over it will be non-blocking.

In our `Unsubscribe` function, we use the helpful `sync.Once` helper from the standard library to ensure we run the function once and only once. We remove the channel from the feed's internal list of listeners and then close our buffered sub channel to wrap things up nicely. Another interesting to note is the function signature for the `Err() <-chan error` func. Specifying `<-chan error` basically tells whomever uses this function that the returned channel is only capable of _receiving_ data, preventing any external sends into the error channel. So far, our first implementation meets most of the invariants we defined above, but it's still not _generic_...let's fix that.

## Next Steps: A Generic Event Feed

![image](https://github.com/MariaLetta/free-gophers-pack/blob/master/illustrations/png/15.png?raw=true)

Although Go itself doesn't have built in generic functions, we can still define functions that take in the empty interface `interface{}`, and use the reflect package from the standard library to perform operations based on the exact type and value passed in. Let's see what we can do by defining a fully generic Feed.

```go
type Feed struct {
    lock sync.Mutex
    sending chan struct{}
    sendCases []reflect.SelectCase
}

func (f *Feed) Subscribe(ch interface{}) (Subscription, error) {
    f.lock.Lock()
    defer f.lock.Unlock()
    val := reflect.ValueOf(ch)
    chanTyp := val.Type()
    if chanTyp.Kind() != reflect.Chan {
        return nil, fmt.Errorf("wanted channel type as input, received %v", chanTyp)
    }
    cs := reflect.SelectCase{Dir: reflect.SelectSend, Chan: val}
    f.sendCases = append(f.sendCases, cs)
    return &sub{
        feed: f,
        channelIndex: len(f.sendCases)-1,
        channel: val,
        err: make(chan error, 1),
    }
}
```

Now, we can accept any generic channel, and instead of keeping track of `[]chan bool`, we now keep track of `[]reflect.SelectCase`, what does this mean? From [Godocs](https://godoc.org/reflect#SelectCase), we see there it's a useful wrapper in the standard library for keeping track of a channel and whether or not it is send-only or receive-only, which will be helpful later.

In our `Send` function, we can now accept _any_ value by attempting to send over the tracked channels:

```go
func (f *Feed) Send(value interface{}) {
    rvalue := reflect.ValueOf(value)
    <-f.sending
    f.lock.Lock()
    for _, cs := range f.sendCases {
        // Set the sent value on all channels.
        cs.Send = rvalue
        cs.TrySend(rvalue)
    }
    f.lock.Unlock()
    f.sending <- struct{}{}
}
```

We use `sending chan struct{}` as a way to protect the Send function and make it fully thread safe, as there could be many calls to Send occurring at the same time, we want to ensure they can only happen sequentially.

## A Critical Problem With Our Approach

It seems like now we met most of our invariants, but what's critically wrong from the code above? It seems simple enough, no? Consider the first section of this blog post where we talked about the merits of unbuffered vs. buffered channels as well as typical gotchas regarding them. Well, when we attempt to subscribe any generic channel into our `Feed` data structure using the code above, we have **no way of determinining whether the channel is buffered or not**. That is, we cannot determine if some operations will block on send while others will not, and that will lead to chaos if we try to use our naïve approach at runtime. Plus, keep in mind we deliver data to listeners by doing a simple for loop over a list of registered channels and performing a send operation, but what if we have the following registered listeners:

```go
[
    make(chan bool, 2),
    make(chan bool),
    make(chan bool, 1)
]
```

Any attempt to do:

```go
cs.TrySend(rvalue)
```

...will succeed on the first channel's send, but **block** the thread on the second if there is no active listener, and remember one of our invariants was that the sender should **not** care about who's listening or their status, but this invariably leads to a major problem.

Can we do better?

## Towards a Smarter, Non-Blocking Approach

Instead of trying once and possibly failing, we _must_ keep trying to send the data over every listener until all of them either:

- (a) succeed, or...
- (b) some of them unsubscribe themselves from the feed

The `TrySend` function can actually fail and returns a bool value of `false` if the channel we are sending to has a full buffer, in which case our logic above does not work. Instead of keeping a single slice of channels we should send over, we can keep two slices: one for channels we have yet to attempt to send, and another for channels we finished sending over. Let's call the former `pendingProcessing` and the other `inProgress`.

```go
type Feed struct {
    lock sync.Mutex
    pendingProcessing []reflect.SelectCase
    inProgress []reflect.SelectCase
}
```

We should have a while loop which proceeds until all inProgress cases have been marked as done, and if some of the receiving channels are blocked, we should wait in each iteration of the loop until they unblock before attempting to send again. Here's a bit of pseudocode of how this could work:

```python
processing = feed.in_progress
while:
    for case in processing:
        # If the channel is blocking, this will fail!
        if case.try_send:
            cases = cases.mark_succeded(case)
    if len(cases) == 0:
        break
    # For cases that failed and were not marked as succeeded, we pseudo-randomly
    # pick one of them, block with a select statement, and wait for it to unblock
    # and receive our data. Then, we mark it as succeeded.
    chosen = wait_for_unblocked_and_send(cases)
    cases = cases.mark_succeded(chosen)
```

![image](https://raw.githubusercontent.com/MariaLetta/free-gophers-pack/master/illustrations/png/19.png)

With the logic above, eventually every single in progress case will complete and will get data sent over its channel. Next, we need a function which will deactivate cases from the `inProgress` slice as we successfully send over them, the `mark_succeeded` function from our pseudocode above.

```go
// removes an channel at index i efficiently as order does not
// matter for listeners we keep track of.
func deactivate(cs []reflect.SelectCase, index int) []reflect.SelectCase {
    last := len(cs) - 1
    cs[index], cs[last] = cs[last], cs[index]
    return cs[:last]
}
```

We name the function `deactivate` as it basically removes an item at an index from the slice of cases we are attempting to send over. Once the send succeeds, we `deactivate` the case.

```go
func (f *Feed) Send(value interface{}) (nsent int) {
    rvalue := reflect.ValueOf(value)
    <-f.sending

    // Add new cases from the pendingProcessing slice after taking the send lock.
    f.lock.Lock()
    f.inProgress = append(f.inProgress, f.pendingProcessing...)
    f.pendingProcessing = nil
    f.lock.Unlock()

    // Set the Send value on all channels.
    for i := firstSubSendCase; i < len(f.inProgress); i++ {
        f.inProgress[i].Send = rvalue
    }

    // Send until all channels have been chosen. 'cases' tracks a prefix
    // of inProgress. When a send succeeds, the corresponding case moves to the end of
    // 'cases' and it shrinks by one element.
    cases := f.inProgress
    for {
        // Try sending without blocking before adding to the select set.
        // This should usually succeed if subscribers are fast enough and have free
        // buffer space.
        for i := 0; i < len(cases); i++ {
            if cases[i].Chan.TrySend(rvalue) {
                nsent++
                cases = cases.deactivate(i)
                i--
            }
        }
        if len(cases) == 0 {
            break
        }
        // Select on all the receivers, waiting for them to unblock.
        chosen, _, _ := reflect.Select(cases)
        cases = cases.deactivate(chosen)
        nsent++
    }

    // Forget about the sent value.
    for i := 0; i < len(f.inProgress); i++ {
        f.inProgress[i].Send = reflect.Value{}
    }
    f.sending <- struct{}{}
    return nsent
}
```

What's going on over here? Well, from the pseudocode from earlier, if `TrySend` fails, we'll go into the latter part of our for loop, which will use `reflect.Select(cases)`, which pretty much simulates a real, blocking `select` block in Go!

```go
select {
case waitTillUnblocked(someChan):
    someChan <- data
}
```

This will pseudorandomly pick one of the channels which did not succeed, block the thread until the channel is unblocked, and then send the data over it. With this, we have all the pieces needed to test this out and make sure it fits our invariants. If we try running

## Testing & Benchmarks

How does one even begin to test something like this? With channels of course! Given we have a lot of concurrency, we will have to be careful with our test setup to avoid any deadlocks or goroutine leaks, all while ensuring we properly test the behavior of our functions. Let's try to setup a test to ensure the one-to-many functionality invariant of our library is met.

```go
func TestFeed_OneToMany(t *testing.T) {
    feed := &Feed{
        pendingProcessing: make([]reflect.SelectCase, 0),
        inProgress:        make([]reflect.SelectCase, 0),
    }

    var done, subscribed sync.WaitGroup
    // We create a wait group for subscribing and a wait group for completing
    // the receipt of the value.
    numRoutines := 10
    done.Add(numRoutines)
    subscribed.Add(numRoutines)
    for i := 0; i < numRoutines; i++ {
        go func() {
            defer done.Done()
            ch := make(chan bool, 0)
            sub, err := feed.Subscribe(ch)
            defer sub.Unsubscribe()
            if err != nil {
                t.Fatal(err)
            }
            // We notify the wait group we finished subscribing.
            subscribed.Done()
            select {
            case val := <-ch:
                t.Logf("Received value in listener %d: %v", i, val)
                return
            case err := <-sub.Err():
                t.Errorf("Received error in listener %d: %v", i, err)
                return
            }
        }()
    }

    // We wait for all subscriptions to be completed before we send anything out.
    subscribed.Wait()
    if nsent := feed.Send(true); nsent != numRoutines {
        t.Errorf("First send delivered %d times, wanted %d", nsent, numRoutines)
    }
    // We wait for the values to be received before finishing the test.
    done.Wait()
}
```

In the example above, we simply setup 10 subscribers which are waiting to receive
information on a subscribed channel they initialized. We create two wait groups so we can
wait for the subscriptions to be registered and for the subscribers to receive the values. We simply do a `feed.Send` and finally the number of sends matches the number of goroutines - simple enough!

Let's see how a benchmark would work here.

```go
func BenchmarkFeed_Send(b *testing.B) {
    feed := &Feed{
        pendingProcessing: make([]reflect.SelectCase, 0),
        inProgress:        make([]reflect.SelectCase, 0),
    }
    numRoutines := 1000

    var wg sync.WaitGroup
    wg.Add(numRoutines)
    for i := 0; i < numRoutines; i++ {
        ch := make(chan int, numRoutines)
        feed.Subscribe(ch)
        go func(cc chan int) {
            for i := 0; i < b.N; i++ {
                <-cc
            }
            done.Done()
        }(ch)
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        if feed.Send(i) != numRoutines {
            b.Fatal("Incorrect number of sends")
        }
    }

    b.StopTimer()
    done.Wait()
}
```

The results only make 1 allocation per operation as we initialize a channel, but we do see quite a few ns/operation of running our `Send` function:

```
$ go test -bench=. -benchmem
goos: darwin
goarch: amd64
BenchmarkFeed_Send-4   	   10000	    252560 ns/op	      13 B/op	       1 allocs/op
PASS
```

We can wrap this up! So we ended up doing a whirlwind tour of buffered vs. unbuffered channels, learned about goroutines and deadlock, and finally ended up recreating the event library used by the [go-ethereum](https://github.com/ethereum/go-ethereum/blob/master/event/feed.go) and the [Prysm](https://github.com/prysmaticlabs/prysm/blob/master/shared/event/feed.go) project, hopefully making concurrency a bit less intimidating, one step at a time :). The finalized event feed library from the prior section looks very similar to the Go-Ethereum `Feed` implementation so feel free to use it as a reference and try to run it yourself!

## Other Recommended Readings

- [Buffered Channels in Go](https://robertbasic.com/blog/buffered-vs-unbuffered-channels-in-golang/)
- [Anatomy of a Go Channel](https://medium.com/rungo/anatomy-of-channels-in-go-concurrency-in-go-1ec336086adb)
