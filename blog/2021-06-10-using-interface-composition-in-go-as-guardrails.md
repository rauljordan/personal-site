---
layout: post
title:  "Using Interface Composition in Go As Guardrails"
preview: Protect yourself using incremental, data access patterns 
image: https://i.imgur.com/ljUXRUz.png
date: 2021-Jun-10
tags: 
  - golang
---

> Composition over inheritance
> - [Someone, somewhere](https://en.wikipedia.org/wiki/Composition_over_inheritance)

Go, as a programming language, favors simplicity. When writing abstractions in Go, interfaces are some of the most powerful tools available to developers, providing a whole suite of useful functionality for your applications and expressive packages.

One underappreciated pattern of using interfaces in Go is the ability to _compose_ them to build up more complex abstractions. Interface composition allows developers to create small building blocks which only expose necessary methods. This pattern is a powerful way of restricting access to dangerous methods and helping to protect developers from biting their own tongue.

![Image](https://i.imgur.com/ljUXRUz.png)

## Brief detour into interface composition

_Skip ahead if you are already familiar with interface composition_

Interfaces are built-in types meant to define _behavior_ of a potential set of types, such as structs. How that behavior is implemented is up to the specific struct that meets the requirements of the interface. The great thing about interfaces in Go is they are _composable_. This means you can build up pretty sophisticated interfaces using basic building blocks. For example, let's say we want to define some interface for a `File` type.

```go
type File interface {
  Read(p []byte) (n int, err error)
  Write(p []byte) (n int, err error)
  Close() error
  Name() string
}
```

Reading and writing bytes somewhere is such a common occurence in Go programs that the [standard library](https://golang.org/pkg/io/#Reader) provides very minimal, yet powerful interfaces to meet these exact needs. Namely: `io.Reader` and `io.Writer`.

```go
type Reader interface {
  Read(p []byte) (n int, err error)
}

type Writer interface {
  Write(p []byte) (n int, err error)
}
```

Sometimes, objects that read also like to write, so we can combine them as follows and reduce the amount of code we have to duplicate:

```go
type ReadWriter interface {
  Reader
  Writer
}
```

There's even a `ReadWriteCloser` interface that composes the primitive `io.Closer` type as well. We can rewrite our file interface by composing basic interfaces as follows:

```go
import "io"

type File interface {
  io.ReadWriteCloser
  Name() string
}
```

Because a file is an `io.ReadWriteCloser`, it can be used by _any Go functions that accept that interface as an argument. This makes it trivial to write tests, integrate into third-party packages, and provides what is close enough to a **standard** for Go projects, especially using the built-in `io` package for code reuse. Interface composition is a powerful tool, especially when working with small interfaces you can combine into more expressive abstractions.

Composition of this flavor is quite different from traditional, object-oriented inheritance. Because our file interface is an `io.ReadWriteCloser`, it is also a `io.Closer`, an `io.Reader`, and `io.Writer`, so there is no shortage of useful places it can be used and immediately integrated with. This is the power of composition over traditional inheritance in other programming languages. Other examples of popular types that implement the `io.Reader` interface are HTTP requests, websocket connections, streams, and more.

## Real-life use case: code guardrails using incremental interface composition

At my company, we maintain an open source implementation of an Ethereum consensus node called [Prysm](https://github.com/prysmaticlabs/prysm). The current Ethereum proof of stake chain secures many billions of dollars, and a large percentage of nodes in the network choose to run our software, meaning we must have the highest quality guarantees and a low margin-of-error.

One particular problem that hurt us several times over the years was allowing unrestricted database access. For example, we had a single `Database` interface that would be define getters and setters for the data we care about at runtime.

```go
type Database interface {
  SaveBlock(block *pb.BeaconBlock) error
  BlockByRoot(root [32]byte) (*pb.BeaconBlock, error)
  SaveState(state *pb.BeaconState, blockRoot [32]byte) error
  StateByRoot(blockRoot [32]byte) (*pb.BeaconState, error)
  ... // A few other critical methods.
}
```

The problem in a large codebase is, although we trust all our teammates to not misuse code, having public APIs such as this interface can be extremely risky. We would pass in this `Database` interface to all services that wanted it, and anyone could call dangerous access methods such as `SaveBlock`. Even accessing methods such as `StateByRoot` excessively would lead to bottlenecks at runtime in memory use. In fact, one consensus failure we had in a local test network was due to us saving states in multiple places, leading to catastrophe.

### Incremental access restrictions: use only what you need

Even though you may trust yourself to use your own code responsibly, new developers and contributors will join your project and will assume any public method is free to use if it helps them solve a real problem. Instead of informally enforcing some arbitrary rules on teammates, we redesigned how we use our `Database` interface for greater safety. We noticed the vast majority of cases only needed **read-access** to certain data types. On top of that, it was rare that we needed **state read-access** as well. We used interface composition to restructure our code below:


```go
type NoStateAccessReadOnlyDB interface {
  BlockByRoot(root [32]byte) (*pb.BeaconBlock, error)
}

type ReadOnlyDB interface {
  NoStateAccessReadOnlyDB
  StateByRoot(blockRoot [32]byte) (*pb.BeaconState, error)
}

type ReadWriteDB interface {
  ReadOnlyDB  
  SaveState(state *pb.BeaconState, blockRoot [32]byte) error
  StateByRoot(blockRoot [32]byte) (*pb.BeaconState, error)
}
```

This is powerful, because then we only pass in _what we need_ to the services that require database access. This becomes easy to audit and ensures that even if someone tries to use dangerous `Save` methods, they will not even have that as an option. They are free to code as they please. We never had any further issues with unrestricted access to database writes after this improvement.

Thanks for reading!
