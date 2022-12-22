+++
title = "Building a Service Registry in Go"
date = 2020-03-10

[taxonomies]
tags = ["golang"]
+++

Thinking of building an application in Go that has multiple running parts? Say you have some server that needs to do a bunch of different things while it runs, such as perform some backround jobs, update caches, handle several requests, expose a REST API, perform outbound requests to other APIs, all without blocking the main thread - what do you do? Typically, this is a good task for creating a microservices architecture where you have multiple applications talking to each other over some network service mesh, each containerized in some nice docker environment, orchestrated through something like Kubernetes or docker-compose.

<!-- more -->

![image](https://golang.org/doc/gopher/fiveyears.jpg)

However, sometimes you just want a straightforward application that can do it all! A good example of this is a blockchain node, such as a Bitcoin or Ethereum node, which needs to do a bunch of things while it runs including:
- Syncing the blockchain
- Exposing an RPC endpoint
- Mining blocks, rewarding miners accordingly
- Listening for p2p connections and handling the lifecycles of peers
- Maintaining an open database connection to some persistent key-value store such as Level-DB

Some of the items above depend on each other, and they should all run when I start a single process for the node. How do we implement something like this in Go? This is a perfect use case for dependency injection. In this blog post, we're going to look at a simple pattern to get this done.

First, our runtime is basically a series of **services**, each doing a bunch of things, asking for or sending data between each other, and possibly having errors or critical failures that we should easily be aware of from a bird's eye view. We want to ideally declare the services that should run upon starting the process, and should have a way of gracefully stopping them if the service dies. We can then define an interface called `Service` which lets us
1. Start the process
2. Stop the process
3. Check the process' current status

Anything that meets the criteria above is a service under our definition! We'll see why this is helpful below.

```go
type Service interface {
	// Start spawns the main process done by the service.
	Start()
	// Stop terminates all processes belonging to the service,
	// blocking until they are all terminated.
	Stop() error
	// Returns error if the service is not considered healthy.
	Status() error
}
```

Next up, we're gonna define an actual struct that will keep track of services by their particular type. We keep around a map of services by their type, but we _also_ keep around an _ordered_ list of these types, given maps in Go do not have a set order. It's important for us to define an order of services, as services can often depend on others that should be initialized **first**.

```go
// ServiceRegistry provides a useful pattern for managing services.
// It allows for ease of dependency management and ensures services
// dependent on others use the same references in memory.
type ServiceRegistry struct {
	services     map[reflect.Type]Service // map of types to services.
	serviceTypes []reflect.Type           // keep an ordered slice of registered service types.
}

// NewServiceRegistry starts a registry instance for convenience
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[reflect.Type]Service),
	}
}
```

Next up, we want to be able to register services into our registry in a particular order. If a service does not exist in the registry, we add it to the map and also to our ordered list of registered service types.

```go
// RegisterService appends a service constructor function to the service
// registry.
func (s *ServiceRegistry) RegisterService(service Service) error {
	kind := reflect.TypeOf(service)
	if _, exists := s.services[kind]; exists {
		return fmt.Errorf("service already exists: %v", kind)
	}
	s.services[kind] = service
	s.serviceTypes = append(s.serviceTypes, kind)
	return nil
}
```

Next up, we want to be able to actually **start** all our services in the order specified at the time of registration. Let's take a look:

```go
// StartAll initialized each service in order of registration.
func (s *ServiceRegistry) StartAll() {
	log.Infof("Starting %d services: %v", len(s.serviceTypes), s.serviceTypes)
	for _, kind := range s.serviceTypes {
		log.Debugf("Starting service type %v", kind)
		go s.services[kind].Start()
	}
}
```
We start each service in a `goroutine` so it does not block the main thread according to its specified `.Start()` method.
When we wish to **gracefully stop** everything, and we call the `.Stop()` function for each service in **reverse order** of registration, checking for errors along the way.

```go
// StopAll ends every service in reverse order of registration, logging a
// panic if any of them fail to stop.
func (s *ServiceRegistry) StopAll() {
	for i := len(s.serviceTypes) - 1; i >= 0; i-- {
		kind := s.serviceTypes[i]
		service := s.services[kind]
		if err := service.Stop(); err != nil {
			log.Panicf("Could not stop the following service: %v, %v", kind, err)
		}
	}
}
```

### So How Do We Use This?

Now we have a cool way to do run multiple services from within a single application, how do we put it to use? Let's talk about a simple architecture!

```
mygoproject/
  p2p/
    service.go
  api/
    service.go
  db/
    service.go
  numbercrunching/
    service.go 
```

We register and start each service in the required order:

```go
package main

func main() {
    registry := NewServiceRegistry()
    
    // Register our database first.
    db := database.InitializeDB()
    registry.RegisterService(db)
    
    // We then start up our p2p server.
    p2pServer := p2p.InitializeP2P()
    registry.RegisterService(p2pServer)

    // We then start up our API.
    apiServer := api.InitializeAPI()
    registry.RegisterService(apiServer)

    // We then start up some number crunching service.
    miscServer := misc.InitializeNumberCrunching()
    registry.RegisterService(miscServer)
    
    // Rev it up!
    registry.StartAll()
}
```

Does the code above do something...? What if my API server depends on the DB, what if my number cruncher depends on my API...? **How can we we implement dependencies between services???**

### Enter Dependency Injection

There's a reason we declared and registered each service in the order specified. That is, some services depend on others, and we want to keep the whole dependency graph quite simple. An important programming paradigm is the idea of **separation of concerns**, which means each module in a program should be concerned with its specific logic and shouldn't be tasked to do things outside of its logical scope. That is, you shouldn't expect your API server to also deal with the internals of handling the db connection, or with dialing other servers via a p2p peer manager. Everything should be self-contained, easy to reason about, and easier to test. 

![image](https://miro.medium.com/max/5000/1*Dqi3QdCy-LbdtS69-rLZcg.png)

A big part of separation of concerns in our toy example above is that each service shouldn't care about how to get access to other services. It should be provided its dependencies at the time of initialization. That is, if I'm the API server, I should just know I **have** access to the db and the p2p services, _I shouldn't need to worry about how to request them fetch them from somewhere far away_.

This concept of explicitly defining the dependencies and _injecting_ them into services that need them is known as **dependency injection**, a fancy term that now makes more sense when you look at our code above. If you look at our API server code, it probably looks quite straighforward if we follow the service pattern above:

```go
package api

type Server struct {
    db *database.Database
    p2pServer *p2p.Server
}
```

The API Server doesn't need to worry about how to access the db or p2p services, as it already has them injected into it upon initialization! Pretty cool...but our service registry code doesn't allow for this injection just yet. Let's see how we can do it.


> Dependency injection is awesome

```go
// FetchService takes in a struct pointer and sets the value of that pointer
// to a service currently stored in the service registry. This ensures the input argument is
// set to the right pointer that refers to the originally registered service.
func (s *ServiceRegistry) FetchService(service interface{}) error {
	if reflect.TypeOf(service).Kind() != reflect.Ptr {
		return fmt.Errorf("input must be of pointer type, received value type instead: %T", service)
	}
	element := reflect.ValueOf(service).Elem()
	if running, ok := s.services[element.Type()]; ok {
		element.Set(reflect.ValueOf(running))
		return nil
	}
	return fmt.Errorf("unknown service: %T", service)
}
```

The fetch service function above is the key. It let's us grab the right pointer to a service we keep track of in our service registry. We can use this for dependency injection. 

Let's refactor our code to use it:

```go
package main

import "log"

func main() {
    registry := NewServiceRegistry()
    
    // Register our database first.
    db := database.InitializeDB()
    registry.RegisterService(db)
    
    // We then start up our p2p server.
    registerP2P(registry)

    // We then start up our API.
    registerAPI(registry)
    
    // Rev it up!
    registry.StartAll()
}

func registerP2P(reg *ServiceRegistry) {
    var dbService *database.Service
    if err := reg.FetchService(&dbService); err != nil {
        log.Fatal(err)
    }

    p2pServer := p2p.InitializeP2P(p2p.Config{
        database: dbService, 
    })
    registry.RegisterService(p2pServer)
}

func registerAPI(reg *ServiceRegistry) {
    var dbService *database.Service
    if err := reg.FetchService(&dbService); err != nil {
        log.Fatal(err)
    }

    var p2pService *p2p.Server
    if err := reg.FetchService(&p2pService); err != nil {
        log.Fatal(err)
    }

    apiServer := api.InitializeAPI(api.Config{
        database: dbService,
        p2p: p2pService, 
    })
    registry.RegisterService(apiServer)
}
```

There we go! We explicitly define the dependencies each service needs upon initialization, making it easy for them to maintain autonomy and separation of concerns accordingly. Next time if you have to choose between creating a complex microservice architecture, consider this simple monolith with dependency injection to save you some headaches!
We actually use this exact same pattern in my team's `Prysm` project, our implementation of the Ethereum 2.0 blockchain Go you can find [here](https://github.com/prysmaticlabs/prysm/blob/8d3fc1ad3ecf5457bb03621f2bbf50022cfd9d65/shared/service_registry.go#L14).
