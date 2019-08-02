# [Go](https://golang.org/) Query Bus
A query bus to fetch all the things.  

[![Build Status](https://travis-ci.org/io-da/query.svg?branch=master)](https://travis-ci.org/io-da/query)
[![Maintainability](https://api.codeclimate.com/v1/badges/7e612d86b1e8bbe89858/maintainability)](https://codeclimate.com/github/io-da/query/maintainability)
[![Test Coverage](https://api.codeclimate.com/v1/badges/7e612d86b1e8bbe89858/test_coverage)](https://codeclimate.com/github/io-da/query/test_coverage)
[![GoDoc](https://godoc.org/github.com/io-da/query?status.svg)](https://godoc.org/github.com/io-da/query)

## Installation
``` go get github.com/io-da/query ```

## Overview
0. [Queries](#Queries)
0. [Handlers](#Handlers)
0. [Result](#Result)
0. [Iterator Handlers](#Iterator-Handlers)
0. [Iterator Result](#Iterator-Result)
0. [Error Handlers](#Error-Handlers)
0. [The Bus](#The-Bus)  
   0. [Tweaking Performance](#Tweaking-Performance)  
   0. [Shutting Down](#Shutting-Down)  
0. [Benchmarks](#Benchmarks)
0. [Examples](#Examples)

## Introduction
This library is intended for anyone looking to query for data in a decoupled architecture. **No reflection, no closures.**

## Getting Started

### Queries
Queries are any type that implements the _Query_ interface. Ideally they should contain immutable data.  
```go
type Query interface {
    ID() []byte
}
```

Queries can optionally implement the _Cacheable_ interface for builtin caching.  
```go
type Cacheable interface {
    CacheKey() []byte
    CacheDuration() time.Duration
}
```

### Handlers
Handlers are any type that implements the _Handler_ interface. Handlers must be instantiated and provided to the bus using the ```bus.Handlers``` function.  
```go
type Handler interface {
    Handle(qry Query, res *Result) error
}
```
Handlers _catch_ the query (stop propagation) whenever they explicitly use ```res.Done()```. Otherwise the query will be provided to all the handlers that expect it. This strategy can be used to have multiple fallback handlers for the same query or have the _Result_ be populated by multiple handlers.  
Whenever a query fails to be handled, the bus will throw an error. **A query is considered handled whenever any data is provided to the result or when the function ```res.Handled()``` is explicitly used.**

### Result
Result is the _struct_ returned from ```bus.Query```. This is where the data fetched will reside.  
The handlers provide the data to the result using the functions ```res.Add``` or ```res.Set```.  
This data can then be retrieved by using the the functions ```res.First``` (to retrieve only the first result) or ```res.All``` (to return the whole data slice).  

### Iterator Handlers
Iterator handlers are any type that implements the _IteratorHandler_ interface. Iterator handlers must be instantiated and provided to the bus using the ```bus.InitializeIteratorHandlers``` function.  
```go
type IteratorHandler interface {
    Handle(qry Query, res *IteratorResult) error
}
```
These behave nearly identical to normal handlers. However there are a couple of differences:  
 - Expect an _IteratorResult_ instead of _Result_.
 - **Can not be cached**.
 
Iterator handlers are intended to be used to with large sets of data. Providing a possibility to iterate over the data without additional preloading.  

### Iterator Result
IteratorResult is the _struct_ returned from ```bus.IteratorQuery```. This struct acts as a proxy between the handlers and the consumer.  
The handlers provide the data to the result using the function ```res.Yield```.  
This data can then be processed while being populated using the the function ```res.Iterate```.  

### Error Handlers
Error handlers are any type that implements the _ErrorHandler_ interface. Error handlers are optional (but advised) and provided to the bus using the ```bus.ErrorHandlers``` function.  
```go
type ErrorHandler interface {
    Handle(qry Query, err error)
}
```
Any time an error occurs within the bus, it will be passed on to the error handlers. This strategy can be used for decoupled error handling.

### The Bus
_Bus_ is the _struct_ that will be used for all the application's queries.  
The _Bus_ should be instantiated (```NewBus()```) and initialized(```bus.InitializeIteratorHandlers```) on application startup.  
The initialization is only required for iterator queries and is separated from the instantiation for dependency injection purposes.  
The application should instantiate the _Bus_ once and then use it's reference for all the queries.  
**The order in which the handlers are provided to the _Bus_ is always respected. This is the order used when propagating queries.**

#### Tweaking Performance
The number of workers for iterator queries can be adjusted.
```go
bus.IteratorWorkerPoolSize(10)
```
If used, this function **must** be called **before** the call to ```bus.InitializeIteratorHandlers```. And it specifies the number of [goroutines](https://gobyexample.com/goroutines) used to handle iterator queries.  
In some scenarios increasing this value can drastically improve performance.  
It defaults to the value returned by ```runtime.GOMAXPROCS(0)```.  
  
The buffer size of the iterator query queue can also be adjusted.  
Depending on the use case, this value may greatly impact performance.
```go
bus.IteratorQueueBuffer(100)
```
If used, this function **must** be called **before** the call to ```bus.InitializeIteratorHandlers```.  
It defaults to 100.  
  
The buffer size of the iterator results channel can also be adjusted.  
Depending on the use case, this value may greatly impact performance.
```go
bus.IteratorResultBuffer(0)
```
If used, this function **may** be called **before** any iterator query is performed.  
It defaults to 0.  

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
| Queries | 201 ns/op |
| IteratorQueries | 783 ns/op |  

Iterator queries add a small overhead and are not worth when used for small sets of data (also due to lack of caching). They are better suited to iterate over large sets of data while avoiding preloading.

## Examples

#### Example Queries
A ```struct``` query.
```go
type Foo struct {
    bar string
}
func (*Foo) ID() []byte {
    return []byte("FOO-UUID")
}
```

A ```string``` query that also implements caching.
```go
type Bar string
func (Bar) ID() []byte {
    return []byte("BAR-UUID")
}
func (Bar) CacheKey() []byte {
    return []byte("BAR-CACHE-KEY")
}
func (Bar) CacheDuration() time.Duration {
    return time.Minute * 5
}
```

#### Example Handlers
A query handler that listens to multiple query types.
```go
type FooBarHandler struct {
}

func (hdl *FooBarHandler) Handle(qry Query, res *Result) error {
    // check the query type
    switch qry := qry.(type) {
    case *Foo:
        // handler logic
        res.Add("Bar")
    case Bar:
        // handler logic
        res.Add("Foo")
    }
    return nil
}
```

An iterator query handler.
```go
type FooBarIteratorHandler struct {
}

func (hdl *FooBarIteratorHandler) Handle(qry Query, res *IteratorResult) error {
    // check the query type
    switch qry := qry.(type) {
    case *Foo:
        // handler logic
        res.Yield("Bar")
    }
    return nil
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
    
    // provide the bus with all of the application's query handlers
    bus.Handlers(
        &FooBarHandler{},
    )
    
    // initialize the bus with all of the application's iterator query handlers
    bus.InitializeIteratorHandlers(
        &FooBarIteratorHandler{},
    )
    
    // query away!
    res1, err := bus.Query(&Foo{})
    // get the first result only
    val := res1.First() // "Bar"

    res2, err := bus.Query(Bar("Bar"))
    // get all the results
    vals := res2.All() // ["Foo"]

    res3, err := bus.IteratorQuery(&Foo{})
    // range over the values, processing them while they are being populated
    for val := range res.Iterate() {
        // do something with the val
        // "Bar"
    }
}
```

## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License
[MIT](https://choosealicense.com/licenses/mit/)