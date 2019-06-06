# [Go](https://golang.org/) Query Bus
A query bus developed with a focus on utility.  

## Installation
``` go get github.com/io-da/query ```

## Overview
1. [Queries](#Queries)
2. [Handlers](#Handlers)
3. [The Bus](#The-Bus)  
   1. [Tweaking Performance](#Tweaking-Performance)  
   2. [Shutting Down](#Shutting-Down)  
4. [Benchmarks](#Benchmarks)
5. [Examples](#Examples)

## Introduction
This library is intended for anyone looking to query for data in a decoupled architecture.  
The _Bus_ will use _workers_ ([goroutines](https://gobyexample.com/goroutines)) to attempt handling the queries in **non-blocking** manner.  
Clean and simple codebase. **No reflection, no closures.**

## Getting Started

### Queries
Queries can be of any type. Ideally they should contain immutable data.  

### Handlers
Handlers are any type that implements the _Handler_ interface. Handlers must be instantiated and provided to the bus on initialization.  
Handlers _catch_ the query (stop propagation) when they return ```true```. This strategy can be used to have multiple fallback handlers for the same query.
```go
type Handler interface {
    Handle(qry Query, resChan chan<- Result) bool
}
```

### The Bus
_Bus_ is the _struct_ that will be used for all the application's queries.  
The _Bus_ should be instantiated and initialized on application startup. The initialization is separated from the instantiation for dependency injection purposes.  
The application should instantiate the _Bus_ once and then use it's reference to trigger all the queries.  
**The order in which the handlers are provided to the _Bus_ is always respected. A query will be _caught_ by the first handler that returns ```true```. At this point the bus will stop propagating the query.**

#### Tweaking Performance
The number of workers can be adjusted.
```go
bus.WorkerPoolSize(10)
```
If used, this function **must** be called **before** the _Bus_ is initialized. And it specifies the number of [goroutines](https://gobyexample.com/goroutines) used to handle concurrent queries.  
In some scenarios increasing the value can drastically improve performance.  
It defaults to the value returned by ```runtime.GOMAXPROCS(0)```.  
  
The buffer size of the query queue can also be adjusted.  
Depending on the use case, this value may greatly impact performance.
```go
bus.QueueBuffer(100)
```
If used, this function **must** be called **before** the _Bus_ is initialized.  
It defaults to 100.  
  
The buffer size of the results channel can also be adjusted.  
Depending on the use case, this value may greatly impact performance.
```go
bus.ResultBuffer(1)
```
If used, this function **may** be called **before** any query is performed.  
It defaults to 1.  

#### Shutting Down
The _Bus_ also provides a shutdown function that attempts to gracefully stop the query bus and all its routines.
```go
bus.Shutdown()
```  
**This function will block until the bus is fully stopped.**

## Benchmarks
The query handler returns a single value for simulation purposes.  

| Benchmark Type | Time |
| :--- | :---: |
| Queries | 589 ns/op |

## Examples

#### Example Queries
A simple ```struct``` query.
```go
type Foo struct {
    bar string
}
```

A ```string``` query.
```go
type FooBar string
```

#### Example Handlers
A query handler that listens to multiple query types.
```go
type FooBarHandler struct {
}

func (hdl *FooBarHandler) Handle(qry Query, resChan chan<- Result) bool {
    // a convenient way to assert multiple query types.
    switch qry := qry.(type) {
    case *Foo, FooBar:
        // handler logic
        resChan <- "bar"
        return true
    }
    return false
}
```

#### Putting it together
Initialization and usage of the exemplified queries and handlers
```go
import (
    "github.com/io-da/query"
)

func main() {
    // instantiate the bus (returns *query.Bus)
    bus := query.NewBus()
    
    // initialize the bus with all of the application's query handlers
    bus.Initialize(
        &FooBarHandler{},
    )
    
    // query away!
    res1, err := bus.Query(&Foo{})
    res2, err := bus.Query(FooBar("foobar"))
}
```

## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License
[MIT](https://choosealicense.com/licenses/mit/)