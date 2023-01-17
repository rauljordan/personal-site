+++
title =  "Go Lessons from Writing a Serialization Library for Ethereum"
date = 2019-07-02

[taxonomies]
tags = ["golang"]

[extra]
photo = ""
+++

```go
type Marshaler interface {
    func MarshalType(
      val reflect.Value, 
      b []byte, 
      lastWrittenIdx uint64,
    ) (nextIdx uint64, error)
}

type Unmarshaler interface {
    func UnmarshalType(
      target reflect.Value, 
      b []byte, 
      lastReadIdx uint64,
    ) (nextIdx uint64, error)
}
```

[Techopedia](https://www.techopedia.com/definition/867/serialization-net) defines serialization as

> The process of converting the state information of an object instance into a binary or textual form to persist into storage medium or transported over a network.

In simple words, it is the task of **translating** data using a common set of rules a standard system can understand, as well as the task of **decoding** translated data back into its original form using the same set of rules. All computers function around the notion of 1s and 0s, otherwise known as bits, which give us a convenient, common format we can serialize _into_.

The process of serializing data is called _Marshaling_, and the process of recovering original data from some serialized bytes is known as _Unmarshaling_. We will be using the two terms extensively throughout this post.

<!-- more -->

## Why Serialize at All?

Although all computers speak the same underlying language of 1s and 0s, data gathered from programs running on those computers might have different forms, such as when the username and password a user inputs on a website, or when you type in a Google search. Google needs to be able to take that information from your browser, and translate it into an efficient _format_ its servers can understand to serve your request. Serialized data doesn't _have_ to be in bytes, in fact, it simply needs to be in a format most efficient for the particular system the data will live in - bytes are just the most convenient format for computers to operate in. 

Back in WWII, we used to serialize data into morse code, as that was the most convenient, common format to transmit via voice radio systems [between ships](http://radiomarine.org)!

## Serialization in Current Software Systems

Serialization is the backbone of data transmission on the Internet. In HTTP, the most common serialization format is JSON (JavaScript Object Notation), making it easy to parse user input and is also incredibly supported by the core tools of JavaScript as a language. JSON's benefits come from readability, standardization, and support, although it is not the most compact nor the most efficient way for software systems to transport data. Another alternative is [BSON](http://bsonspec.org/), standing for Binary JSON, which has more efficient serialization rules than JSON and is more flexible for usage as a binary data transmission format.

For real world, scalable systems, companies such as Google opt for the much more robust [Protocol Buffer](https://developers.google.com/protocol-buffers/) specification, another type of serialization language. Protocol Buffers, or protobufs, are used for their powerful **schema-like** property, which allows projects to specify data structures in a common proto language, which can then be used generate data structures across all types of programming languages which are compliant with the protobuf serialization rules.

A common use-case for protobufs is for [Remote Procedure Calls](https://en.wikipedia.org/wiki/Remote_procedure_call), known as RPC calls, which are used data exchange between servers. Say that Google has their ads servers running Java code and they want to feed in data into some other server running Python code for some machine learning analysis on users' ad clicks. Google can specify a common protobuf schema:

```go
message AdClick {
    string url = 1;
    uint64 user_id = 2;
}

message AdClickResponse {
    bool ok = 1;
}

service AdClickAnalyzer {
    rpc SendClickData(AdClick) returns (AdClickResponse)
}
```

And generate the appropriate, compliant data structures in both Python and Java that implement the protobuf serialization format. Having this common language makes data transport simple and unambiguous. Imagine the alternative approach of every project at a company having different rules for how you should communicate with their servers - it would be a nightmare!

## Rule #1 Of Serialization: Don't Reinvent the Wheel

To say serialization has been overdone is an understatement. Such a core aspect of today's Internet has been widely studied, implemented, and optimized to serve the needs of billions of people around the world - every second, any moment. JSON, protobufs, and other specifications are constantly being micro-optimized and maturing to better serve our highly connected world. Creating a _new_ serialization library from scratch is, in general, a bad idea save for very select situations.

Any new approach should leverage existing work and understand the limitations and learnings previous approaches have uncovered. Most of these libraries follow a few core tenets:

### 3 Tenets of Serialization

- Serialization must be unambiguous
- Serialization must be agnostic
- Serialization must be efficient

### Serialization Must Be Unambiguous

Imagine you look at a bunch of raw bytes - something like:

```
01010100100101000100101010101010
```

and imagine you have some serialization/deserialization rules that tell you how to decode this information into some readable, useful data. A serialization function would be completely broken if it were not _injective_. That is, two different object must never map into the same serialization, as this makes the algorithm ambiguous. If the rules tell us the data can decode into the value `[3, 4, 5]` but it turns out the list can also decode as `[5, 4, 3]`, we have no way of knowing which is the correct one! Serialization is stateless, as we have no access to determining the original value that was serialized into aside from using the inverse of the serialization rules to decode it. Serialization _must_ be unambiguous.

### Serialization Must Be Agnostic

From its earlier definition, serialization is simply a set of rules that allow us to encode and decode data into some common format a system can understand. It should **not** make assumptions about underlying data and it should not make explicit implementation decisions in its schema. An example of this is making design decisions around specific types that may not be available across different programming languages, preventing others wanting to use the serialization rules in an implementation from doing so due to language constraints. The **goal** of serialization is simply concerned with translation and definining an unambiguous set of rules for marshaling and unmarshaling data - nothing more, nothing less. Keep it abstract, make no assumptions for what the serialization will be used for, and keep it concise.

### Serialization Must Be Efficient

The purpose of serialization is typically for transporting information from one place to another in a common format. Bits are the smallest type possible for computers to compactly represent information, and they translate well to data transmission protocols that exist today via the wires that power the Internet. It is still possible to create a set of serialization rules, however, that add unnecessary bloat to encoded data.

The rules for marshaling data should _only include what's necessary_ for a decoder to unambigiously unmarshal the bytes back to their original form - nothing more, nothing less. Think of it like the core axioms of mathematics - they are as concise and abstract as possible such that all of math can be derived from those simple rules we hold as true. We could, for example, say the pythagorean theorem is an axiom of math without needing to prove it. In fact, we could just say all theorems are axioms and are true by default - but that would defeat the purpose of math itself and its ruleset! Having more rules than necessary will simply add bloat to a system, possibly leading to massive inefficiencies when that data needs to be transported between machines.

> Keep the core rules simple, don't add more rules than you need to 

# The Task: Implement a New Serialization Algorithm in Go

Now that we've looked at what serialization is and why it matters, let's get into the core of this post. The goal at hand is to create a compliant, [Golang](https://golang.org) implementation of the [Simple Serialize](https://github.com/ethereum/eth2.0-specs/blob/dev/specs/simple-serialize.md) specification created by the official [Ethereum](https://ethereum.org) research team. Ethereum is a global, distributed network of nodes that runs decentralized applications. That is, anyone can use it to write computers that are not ran by any individual, entity, or corporation in a permissionless fashion. Similar to Bitcoin, Ethereum uses a distributed ledger known as a [blockchain](https://en.wikipedia.org/wiki/Blockchain) to maintain a state of the world and reach [consensus](https://en.wikipedia.org/wiki/Consensus_(computer_science)).

The way nodes across the network communicate is by sending packets of byte encoded data via [peer-to-peer (p2p) networking](https://en.wikipedia.org/wiki/Peer-to-peer) using some standard serialization algorithm. Ethereum is currently in the process of upgrading to its 2.0 version, which will contain radical improvements to its core protocol as well as its network architecture. Serialization for Ethereum 2.0 is being revamped, with the Simple Serialize algorithm, **known as SSZ**, having won as the _de facto_ standard for marshaling consensus data into bytes.

Ethereum 2.0 has various in-progress implementations in various languages, and my team, [Prysmatic Labs](https://prysmaticlabs.com), works on a Go implementation called [Prysm](https://github.com/prysmaticlabs/prysm).

Our goal is to create a serialization library in Go so we can easily plug into Prysm that is conforms to standard tests handed to us by Ethereum researchers. This is part of a greater interoperability effort so different implementations can communicate between each other unambiguously.

## Why a New Serialization Format? Why Not Protobuf, JSON, or Others?

Serialization plays a very important role in distributed systems, particularly blockchain-based protocols such as Bitcoin and Ethereum. The entire crux of the network depends on a distributed set of nodes around the world reaching **consensus** on a set of transactions, marking them as canonical based on a set of voting-majority rules. Imagine if you wanted 1000 different servers around the world agreeing on the following data structure between each other:

```go
type State struct {
    AccountBalancesByUserId map[uint64]uint64
}
```

Imagine this information is _so critical_, you can't possibly afford its internal data differing across server implementations. The rules for marshaling and unmarshaling this data **must** be unambiguous. Serialization schemas such as Protocol Buffers provide support for map data structures as seen above, but map support across languages differ underneat the hood. The problem with using protocol buffers for consensus specifically for map-based types is that they **do not provide determinism** between languages. That is, a Java protobuf for that data could marshal/unmarshal differently than a Go implementation, and that is unacceptable.

Serialization for distributed systems operates under an extra constraint that there is zero room for error or non-determinism. For Ethereum, we decided not to use protobufs for serializing important consensus values, as we needed something a lot simpler. This is how simple serialize came to be.

## How Simple Serialize Works

Every serialization specification begins by defining the basic, core types it can support. Basic types then serve as building blocks for more complex types which may have different serialization rules depending on their structure. Once again, a core tenet of serialization is to be _unambiguous_. That is, we need to the serialized bytes of some initial data give us enough information by themselves along with the serialization ruleset to successfully retrieve the original object if we wanted to. The official Simple Serialize specification as of the time of writing is located [here](https://github.com/ethereum/eth2.0-specs/blob/v0.7.1/specs/simple-serialize.md) if you want to follow along.

Simple serialize supports the following basic types:

```
uint8 or byte
uint16
uint32
uint64
bool
```

### Why Not More Basic Types!?

Notice the omission of ints and floats from the above, as they are not used for precision reasons in a consensus-critical system such as Ethereum, which demands absolute determinism. Instead, we keep it simple - using only unsigned integers and booleans. 

SSZ as a serialization tool should only be concerned with agnostic marshaling/unmarshaling of data - it should _not_ be concerned with making assumptions about its use cases. 

[Postel's Law](https://en.wikipedia.org/wiki/Robustness_principle) holds true for SSZ in particular:

> Be conservative in what you do, be liberal in what you accept from others
> 
> TCP/IP Protocol

That is, our SSZ implementation should be able to handle all sorts of complex structures and fail gracefully if they are not serializable. At the same time, it should be conservative in its scope, only doing what it needs to do based on its specification without making assumptions about its inner data or use-cases.

Additionally, adding other types such as BitLists or BitVectors as primitives to Simple Serialize is like adding more axioms to mathematics, it can make your life easier in the short term but makes your abstractions a lot weaker!

### Composite Types

From these basic types, Simple Serialize defines _composite_ types which are:

1. Containers: heterogenous collections of values `{uint64, bool, uint8, uint16}`
2. Vectors: ordered, _fixed_ size collections of the same type of value `[uint64, N]`
3. Lists: ordered, _variable_ length collections of the same type of value `[uint64]`

Two other critical rules of Simple Serialize are the definition of element sizes. An element is said to be _variable size_ if, recursively, it contains at least one element which is of _variable size_. Otherwise, an element is determined to be fixed-size.

Let's look at some examples:

1. `[[uint64], N]` is variable-sized, as despite being a fixed-size vector, its inner element is an unbounded list type
2. `{field1: [uint64, N], field2: bool}` is a fixed-size container as it only contains fixed-size elements
3. `[{field1: uint64, field2: bool}, N]` is fixed-size, as it contains no variable-sized elements

Easy - let's keep going. Now, we define the _marshaling_ rules for each of these basic types, fixed-size types, and variable-size types.

For `uint` types:

```python
assert N in [8, 16, 32, 64, 128, 256]
return value.to_bytes(N // 8, "little")
```

For `bool` types:
```python
assert value in (True, False)
return b"\x01" if value is True else b"\x00"
```

For fixed-size type vectors, we simply serialize them element-wise based on their corresponding elements' marshaling rules.

```python
buffer = []
for item in value:
    buffer.append(marshal(item))
```

For byte-vectors, we simply leave them as they are as they already represent the format we are using for serialization, namely, a collection of bytes.

When it comes to variable-sized items, however, that's when things get interesting. We have **no way of knowing** how big a variable-sized element will be unless we have some delimeters that tell us so in their encoded representation. Think of these values kind of like positional markers of when a variable-sized element begins in a collection of bytes. These values are referred to as _offsets_ in the Simple Serialize specification. 

SSZ offsets are denoted by `uint32` values which serve as positional markers of where variable-sized items occur in a marshaled set of bytes. Take the container type denoted by:

```go
type Example struct {
    Field1 []byte
    Field2 []byte
}

example := &Example{
    Field1: []byte{1, 2},
    Field2: []byte{3},
}
```

Its corresponding SSZ encoding will be:

```
[8 0 0 0 10 0 0 0 1 2 3]
```

Where `8 0 0 0` denotes a uint32 value of 8 which which tells us the first variable sized element begins at index 8 in the list. Similarly, the next offset, `10 0 0 0`, tells us there is another variable-sized element which begins at index 10. When we attempt to unmarshal the bytes into a new instance of type `&Example`, we know exactly where the first field and the next field begin, giving us an unambiguous unmarshaling.

In a complex SSZ type, however, variable-sized elements can be interwoven with fixed-size elements, and we similarly use the idea of offsets to denote where the variable ones occur. Let's see an example:

```go
type Example2 struct {
    Field1 []byte
    Field2 uint16
    Field3 []byte
}

example2 := &Example2{
    Field1: []byte{1, 2},
    Field2: 7,
    Field3: []byte{3},
}
```

This new SSZ encoding, which has a fixed-size `uint16` field mixed in, will look like:

```
[10 0 0 0 7 0 12 0 0 0 1 2 3]
```

Once again telling us that there is a variable-sized element at index 10, one at index 12, and in the middle we have the fixed-size, `uint16` field unambiguously. That's pretty much it! Using these basic rules, one may derive even complex, nested types of fixed-size and variable-sized containers and lists.

## The Limitations of Golang

Go in particular is an interesting language to build a serialization library in due to its lack of generics - a very common trait of other languages. The lack of generics makes it really difficult to write functions that work on general types in Go, and instead Go has to rely on tricky approaches such as inspecting the metadata of values, which can lead to massive inefficiencies. One of the only safe options Go has for writing generic libraries is the ["reflect"](https://blog.golang.org/laws-of-reflection) package, which gives us a whole suite of utilities for doing [metaprogramming](https://en.wikipedia.org/wiki/Metaprogramming) - the notion of a computer program inferring details about its own structure.

The problem with using reflect in Go comes down to its speed. Go was not built in its current form to support generics underneath the hood, making it expensive to infer and operate on abstract types. The closest thing Go has to a generic type is the `interface{}` type, which is not _exactly_ like the generic types in other languages. In Python, for example, which is a dynamically-typed language, the following code is perfectly valid:

```python
def add(x, y):
    return x + y
```

In Go, however, you can't just add two interfaces unless you coerce them into a concrete type that allows for addition, such as uint64. The following piece of Go code is invalid:

```go
func add(v1 interface{}, v2 interface{}) interface{} {
    return v1 + v2
}
```

Giving us the error `invalid operation: v1 + v2 (operator + not defined on interface)`, as expected. If we really want to be able to create a generic add function as above, we'll have to infer the concrete types of the function arguments using the reflect package and act accordingly. Let's use a switch statement to do so:

```go
import (
    "errors"
    "reflect"
)

func add(v1 interface{}, v2 interface{}) (interface{}, error) {
    rval1 := reflect.ValueOf(v1)
    rval2 := reflect.ValueOf(v2)
    k1 := rval1.Type().Kind()
    k2 := rval2.Type().Kind()
    switch {
    case k1 == reflect.Uint64 && k2 == reflect.Uint64:
        return rval1.Interface().(uint64) + rval2.Interface().(uint64), nil
    case k1 == reflect.Uint32 && k2 == reflect.Uint32:
        return rval1.Interface().(uint32) + rval2.Interface().(uint32), nil
    case k1 == reflect.Uint16 && k2 == reflect.Uint16:
        return rval1.Interface().(uint16) + rval2.Interface().(uint16), nil
    case k1 == reflect.Uint8 && k2 == reflect.Uint8:
        return rval1.Interface().(uint8) + rval2.Interface().(uint8), nil
    case k1 == reflect.Int64 && k2 == reflect.Int64:
        return rval1.Interface().(int64) + rval2.Interface().(int64), nil
    case k1 == reflect.Int32 && k2 == reflect.Int32:
        return rval1.Interface().(int32) + rval2.Interface().(int32), nil
    case k1 == reflect.Int16 && k2 == reflect.Int16:
        return rval1.Interface().(int16) + rval2.Interface().(int16), nil
    default:
        return nil, errors.New("expected numeric values")
    }
}
```

Ouch - look at how painful that was! Even worse, let's see how slow it is against a function that just concretely takes in `uint64` values as arguments.

```
BenchmarkAddGeneric-4                   20000000                85.1 ns/op            24 B/op          3 allocs/op
BenchmarkAddUint64-4                    2000000000               0.34 ns/op            0 B/op          0 allocs/op
```

Using reflect makes a big difference - imagine if we have to do the same thing but for more complex types such as nested structs! Let's see how we can tackle actually implementing SSZ in Go.

# High Level Approach

Alright, so we do _not_ have access to generics for our Simple Serialize implementation, and our approach must be generic enough to handle extremely complex data structures efficiently and correctly. Moreover, we need to be able to **round trip, benchmark, and ensure correctness** of our code as we go along. Where do we start? Let's first go back to the basics of how we made a generic `add` function in Go. It is not _truly_ generic, as it actually does type assertions on a few, selected numeric types and applies their addition operations. Along the same vein, our SSZ implementation should only concern itself with supporting the types enumerated in its official [specification](https://github.com/ethereum/eth2.0-specs/blob/v0.7.1/specs/simple-serialize.md). 

The most naive approach would be to have a giant, recursive function that uses a huge switch statement to figure out how to handle every different type. Our approach will instead focus on creating specific marshaler and unmarshaler functions for the various types which satisfy a _common Go interface_.

In general, we need a few things in our marshaler and unmarshaler functions.

- For Marshaling, we want to track the latest positional index we have written into some bytes buffer
- For Unmarshaling, we want to track the latest positional index we have read from some bytes buffer
- Inside both functions, we want to either write (Marshal) or read (Unmarshal) N bytes from a bytes buffer and return the positional index the next Marshaler/Unmarshaler should start from, i.e. the current index + N

With that in mind, we can define the following two key interfaces that are enough for us to handle the entire SSZ spec in an abstract fashion:

```go
type Marshaler interface {
    func MarshalType(val reflect.Value, b []byte, lastWrittenIdx uint64) (nextIdx uint64, error)
}

type Unmarshaler interface {
    func UnmarshalType(target reflect.Value, b []byte, lastReadIdx uint64) (nextIdx uint64, error)
}
```

The `Marshaler` type specifies a function, `MarshalType`, which can take in an abstract value, a buffer we will write into, and the last index into the buffer we have written into. 

> This last index is a positional marker that tells us where we are in the process of 
> writing the serialized encoding of the input value

Similarly, the `Unmarshaler` type specifies a function, `UnmarshalType`, which can take a buffer of SSZ-marshaled bytes, a target value we will deserialize _into_, and the last index we have read from the buffer. This last index is a positional marker that tells us where we are in the process of reading marshaled bytes from the serialized encoding.

## Marshaling

The first thing we should do when Marshaling is to go back to our trusty switch statement to determine the type we're looking at before we decide how it should be handled. 

```go
func Marshal(data interface{}) ([]byte, error) {
    val := reflect.ValueOf(data)
    m, err := makeMarshaler(val)
    if err != nil {
        return nil, err
    }
    totalBufferSize := determineSize(val)
    buffer := make([]byte, totalBufferSize)
    if _, err = m.MarshalType(rval, buf, 0 /* start index */); err != nil {
        return nil, fmt.Errorf("failed to marshal for type: %v", rval.Type())
    }
    return buffer, nil
}

func makeMarshaler(val reflect.Value) (Marshaler, error) {
    kind := val.Type().Kind()
    switch {
    case kind == reflect.Bool:
        // Handle bool marshaling...
    case kind == reflect.Uint8:
        // Handle uint marshaling...
    case kind == reflect.Uint16:
        // Handle uint marshaling...
    case kind == reflect.Uint32:
        // Handle uint marshaling...
    case kind == reflect.Uint64:
        // Handle uint marshaling...
    case kind == reflect.Slice && typ.Elem().Kind() == reflect.Uint8:
        // Handle byte slice marshaling...
    case kind == reflect.Array && typ.Elem().Kind() == reflect.Uint8:
        // Handle byte array marshaling...
    case isBasicTypeSlice(val.Type()) || isBasicTypeArray(val.Type()):
        // Handle basic-type array/slice marshaling...
    case kind == reflect.Array && !isVariableSizeType(typ.Elem()):
        // Handle fixed-size element array marshaling...
    case kind == reflect.Slice && !isVariableSizeType(typ.Elem()):
        // Handle fixed-size element slice marshaling...
    case kind == reflect.Slice || kind == reflect.Array:
        // Handle variable-sized element slice/array marshaling...
    case kind == reflect.Struct:
        // Handle struct marshaling...
    case kind == reflect.Ptr:
        // Handle pointer marshaling...
    default:
        return nil, fmt.Errorf("type %v is not serializable", typ)
    }
}

// determineSize figures out the total size of the SSZ-encoding in order
// to preallocate a buffer before marshaling.
func determineSize(val reflect.Value) uint64 {
    ...
}
```

On a case-by-case basis, we can return an abstract `Marshaler` interface for the different types we support for SSZ. Notice at the top, we have to also determine the _total size_ of the serialized bytes before we begin writing to a buffer, otherwise we'll be getting nasty index out of range issues in Go. That is, if we want to encode the number `uint64(5)`, we need 8 bytes to do so, so we would need to preallocate some fixed-size buffer before we begin marshaling at all.

The above gives us as generic and as flexible of a basic marshaling approach for SSZ. To proceed, we'd just need to fill in the blanks and create marshalers that work for the different types based on the SSZ specification. Here's what the `MarshalType` function for some of these basic types would look like:

```go
import "binary"

type BoolSSZ bool
type Uint64SSZ uint64

// Boolean basic type marshaling.
func (b BoolSSZ) MarshalType(val reflect.Value, b []byte, lastWrittenIdx uint64) (uint64, error) {
    if val.Bool() {
        buf[lastWrittenIdx] = uint8(1)
    } else {
        buf[lastWrittenIdx] = uint8(0)
    }
    return lastWrittenIdx + 1, nil
}

// Uint64 basic type marshaling.
func (b Uint64SSZ) MarshalType(val reflect.Value, b []byte, lastWrittenIdx uint64) (uint64, error) {
    v := val.Uint()
    encoded := make([]byte, 8)
    binary.LittleEndian.PutUint64(encoded, uint64(v))
    copy(buf[lastWrittenIdx:lastWrittenIdx+8], encoded)
    return lastWrittenIndex + 8, nil
}
```

As long as we satisfy the Marshaler interface, we can rinse and repeat for all basic types! Go is indeed powerful, but let's take a look at the most complicated marshaling example.

```go
var BytesPerLengthOffset = 4

type SliceMarshaler struct {
    elementMarshaler Marshaler
}

func NewSliceMarshaler(val reflect.Value) (*SliceMarshaler, error) {
    // We attach the marshaler for the slice's particular element type
    // to the struct for usage later.
    elementMarshaler, err := makeMarshaler(val.Elem())
    if err != nil {
        return nil, err
    }
    return &SliceMarshaler{elementMarshaler}
}

func (s *SliceMarshaler) MarshalType(val reflect.Value, b []byte, lastWrittenIdx uint64) (uint64, error) {
    currentIndex := lastWrittenIndex
    typ := val.Type()
    var err error
    if isVariableSizeType(typ) {
        currentIndex, err = s.marshalVariableSize(val, b, currentIndex)
        if err != nil {
            return 0, err
        }
    } else {
        currentIndex, err = s.marshalFixedSize(val, b, currentIndex)
        if err != nil {
            return 0, err
        }
    }
    return index, nil
}

// If an element is variable sized, we need to also keep track of its 
// offset indices and where they tell us to begin checking for its 
// next variable-sized element in the input. Using these two positional
// trackers, we can write the offsets at their respective positions
// as well as write the actual values we are marshaling into the buffer
// at the current index we are tracking.
func (s *SliceMarshaler) marshalVariableSize(val reflect.Val, b []byte, idx uint64) (uint64, error) {
    currentIndex := idx
    fixedElementIndex := currentIndex
    startOffset := lastWrittenIndex
    currentOffsetIndex := startOffset + val.Len()*BytesPerLengthOffset
    nextOffsetIndex := currentOffsetIndex
    // If the elements are variable size, we need to include offset indices
    // in the serialized output list.
    for i := 0; i < val.Len(); i++ {
        nextOffsetIndex, err = s.elementMarshaler.MarshalType(val.Index(i), b, currentOffsetIndex)
        if err != nil {
            return 0, err
        }
        // Write the variable-sized element offset.
        offsetBuf := make([]byte, BytesPerLengthOffset)
        binary.LittleEndian.PutUint32(offsetBuf, uint32(currentOffsetIndex-startOffset))
        copy(b[fixedElementIndex:fixedElementIndex+BytesPerLengthOffset], offsetBuf)

        // We increase the positional index tracker of the offsets accordingly.
        currentOffsetIndex = nextOffsetIndex
        fixedElementIndex += BytesPerLengthOffset
    }
    return currentOffsetIndex
}

// If each element is not variable size, we simply encode sequentially and write
// into the buffer at the last index we wrote at.
func (s *SliceMarshaler) marshalFixedSize(val reflect.Val, b []byte, idx uint64) (uint64, error) {
    currentIndex := idx
    var err error
    for i := 0; i < val.Len(); i++ {
        currentIndex, err = s.elementMarshaler.MarshalType(val.Index(i), b, currentIndex)
        if err != nil {
            return 0, err
        }
    }
    return currentIndex, nil
}
```

Although it looks a bit intimidating, let's break it down into pieces. First, we create an instance of a slice marshaler that also contains a field which gives us access to a marshaler for that slice's individual elements. Then, we check if the slice has fixed-sized elements, for which we simply loop through the slice's length and call the element marshaler at each index of the slice.

Otherwise, we perform a more complex, variable-sized element marshaling. As mentioned in how SSZ works, we need to keep track of particular _offset_ values which tell us the positional indices where variable sized elements begin in the serialization. We also need to keep track of the current writing position into the buffer accordingly. We loop through the slice's length, determine any offsets, write those offsets to the buffer, and update the positional trackers respectively. Voilà, that's pretty much the hardest part of writing an SSZ marshaler :). For container types, such as structs, we have to keep track of the marshalers for each of their fields, and simply loop over the struct's fields and perform the same logic as above.

## Unmarshaling

Now for the hard part...unmarshaling. This part of SSZ is particularly tricky because there are no hard-and fast rules for doing so aside from "doing the inverse of serialization". That is, we'll find ourselves in some hairy situations due to variable-size offsets that will massively increase the complexity of the implementation. Additionally, we'll hit some important gotchas of using Go's reflect package along the way.

Unmarshaling follows the similar philosophy of creating different types that comply with the `Unmarshaler` interface and pattern matching types through a switch statement at the start. Compared to marshaling, however, we do not need to create a bytes buffer of a certain size beforehand, but we have to do something even more difficult. We have to _deeply hydrate_ the value we are unmarshaling into, and that's where the Go gotchas come into play.

We start with our top-level `Unmarshal` function:

```go
func Unmarshal(input []byte, val interface{}) error {
    rval := reflect.ValueOf(val)
    if rval.IsNil() {
        return errors.New("cannot unmarshal into untyped, nil value")
    }
    rtyp := rval.Type()
    // val must be a pointer, otherwise we refuse to unmarshal.
    if rtyp.Kind() != reflect.Ptr {
        return errors.New("can only unmarshal into a pointer target")
    }
    m, err := makeUnmarshaler(val)
    if err != nil {
        return nil, err
    }
    if _, err = m.UnmarshalType(rval.Elem(), input, 0 /* last read idx */); err != nil {
        return fmt.Errorf("could not unmarshal input into type: %v, %v", rval.Elem().Type(), err)
    }
    return nil
}
```

Let's take a look at some basic type unmarshaling:

```go
import "binary"

type Uint32SSZ uint64

func (u Uint32SSZ) UnmarshalBytes(val reflect.Value, b []byte lastReadIdx uint64) (uint64, error) {
    nextReadIdx := lastReadIdx + 4
    buf := make([]byte, 4)
    copy(buf, b[lastReadIdx:nextReadIdxjj])
    val.SetUint(uint64(binary.LittleEndian.Uint32(buf)))
    return nextReadIdx, nil
}
```

So what's the hard part? Let's check out how we would unmarshal the following:

```go
type FixedSizeItem struct {
    Field1 uint32
}

data := [3]*FixedSizeItem{
    &FixedSizeItem{ Field1: 7 },
    &FixedSizeItem{ Field1: 8 },
    &FixedSizeItem{ Field1: 9 },
}

encoded, err := ssz.Marshal(data)
if err != nil {
    panic(err)
}

var decoded [3]*FixedSizeItem
if err := ssz.Unmarshal(encoded, &decoded); err != nil {
    panic(err)
}
```

Given it's a simple array of fixed size elements, we _could_ simply unmarshal element-wise as follows:

```go
currentIndex := lastReadIdx
for i := 0; i < val.Len(); i++ {
    // if val.Index(i).Kind() == reflect.Ptr {
    //     instantiateConcreteTypeForElement(val.Index(i), typ.Elem().Elem())
    // }
    currentIndex, err = m.UnmarshalType(val.Index(i), b, currentIndex)
    if err != nil {
        return 0, fmt.Errorf("failed to unmarshal element of array: %v", err)
    }
}
return currentIndex, nil
```

But if we run this, we get the the following error!
```
panic: cannot call Type() of nil value
```

What gives? When we initialize a variable which will be the target of the unmarshaling, it knows it has to be of type `[3]*FixedSizeItem`, however, we haven't _deeply_ initialized the values of that array with the correct zero values we will unmarshal into. That is, once we're attempting to unmarshal each `&FixedSizeItem` type contained in this array, that item has not been initialized, so what we are feeding the unmarshaler is an array of nil values

```
[nil, nil, nil]
```

Even though the array above has the correct type, we are not able to deduce the unmarshaler properly for each individual element, as they are all initialized to nil! Instead, we have to deeply _hydrate_ the type if it is nil.

```go
func instantiateConcreteTypeForElement(val reflect.Value, typ reflect.Type) {
    val.Set(reflect.New(typ))
}

...

currentIndex := lastReadIdx
for i := 0; i < val.Len(); i++ {
    if val.Index(i).Kind() == reflect.Ptr {
        instantiateConcreteTypeForElement(val.Index(i), typ.Elem().Elem())
    }
    currentIndex, err = m.UnmarshalType(val.Index(i), b, currentIndex)
    if err != nil {
        return 0, fmt.Errorf("failed to unmarshal element of array: %v", err)
    }
}
return currentIndex, nil
```

This makes our unmarshaling logic at a tad bit obscure, but it gets even worse once we have to deal with slices, as slices have no fixed-size as part of their type definition, making it impossible to iterate over them and unmarshal pointwise like we can do with fixed-size arrays. Instead, on each iteration, we have to:

1. Grow the slice's length by 1
2. Deeply hydrate the element at the current index in the slice

```go
func growConcreteSliceValue(val reflect.Value, typ reflect.Type, length int) {
    newVal := reflect.MakeSlice(typ, length, length)
    reflect.Copy(newVal, val)
    val.Set(newVal)
    if val.Index(length-1).Kind() == reflect.Ptr {
        instantiateConcreteTypeForElement(val.Index(length-1), typ.Elem().Elem())
    }
}
```

We can accomplish this using the helper above, but all we're doing once again is instantiating the correct _zero-values_ of a type so we can successfully unmarshal without panics.

Let's put this together and see how we can handle variable-sized slices:

```go
var BytesPerLengthOffset = 4

type SliceUnmarshaler struct {
    elementUnmarshaler Unmarshaler
}

func NewSliceUnmarshaler(val reflect.Value) (*SliceUnmarshaler, error) {
    // We attach the marshaler for the slice's particular element type
    // to the struct for usage later.
    typ := val.Type()
    elementUnmarshaler, err := makeUnmarshaler(typ.Elem())
    if err != nil {
        return nil, err
    }
    return &SliceUnmarshaler{elementUnmarshaler}
}

func (s *SliceUnmarshaler) UnmarshalType(val reflect.Value, b []byte, lastReadIdx uint64) (uint64, error) {
    if len(input) == 0 {
        newVal := reflect.MakeSlice(val.Type(), 0, 0)
        val.Set(newVal)
        return 0, nil
    }
    growConcreteSliceType(val, typ, 1)
    endOffset := uint64(len(input))
    currentIndex := lastReadIdx
    nextIndex := currentIndex
    offsetVal := input[startOffset : startOffset+BytesPerLengthOffset]
    firstOffset := startOffset + uint64(binary.LittleEndian.Uint32(offsetVal))
    currentOffset := firstOffset
    nextOffset := currentOffset
    i := 0
    for currentIndex < firstOffset {
        nextIndex = currentIndex + BytesPerLengthOffset
        // We have to keep track of the position of each offset we use for reading.
        if nextIndex == firstOffset {
            nextOffset = endOffset
        } else {
            // We then determine the next offset by reading 4 bytes (BytesPerLengthOffset).
            nextOffsetVal := input[nextIndex : nextIndex+BytesPerLengthOffset]
            nextOffset = startOffset + uint64(binary.LittleEndian.Uint32(nextOffsetVal))
        }
        // We grow the slice's size to accommodate a new element being unmarshaled.
        growConcreteSliceType(val, typ, i+1)
        if _, err := s.elementUnmarshaler.UnmarshalType(val.Index(i), input[currentOffset:nextOffset], 0 /* last read element idx */); err != nil {
            return 0, fmt.Errorf("failed to unmarshal element of slice: %v", err)
        }
        i++
        currentIndex = nextIndex
        currentOffset = nextOffset
    }
    return currentIndex, nil
```

We have to add a special caveat that if we are unmarshaling a nil value, we simply create a slice of size 0 and return, otherwise we will encounter painful panics down the line! Phew, using Go and its lack of generics adds massive complexity when it comes to populating a raw pointer of a certain type, but that covers the basics.

### Limitations of Go-SSZ

Unfortunately, there are still some inherent limitations in the rules of SSZ that prevent us from unambiguously unmarshal certain types of data. In particular, consider we run into a value that our unmarshaler determines is empty and is of slice type. We have no way to know if it should unmarshal as `nil` or as `[]`

> SSZ unmarshaling is ambiguous when it comes to empty values in Go

Say if you want to round-trip test the following:

```go
import (
    "reflect"
    "testing"
    "github.com/prysmaticlabs/go-ssz"
)

type AmbiguousItem struct {
    Field1 []byte
    Field2: uint64
}

func TestRoundTrip(t *testing.T) {
    item := &AmbiguousItem{
        Field2: 5,
    }

    encoded, err := ssz.Marshal(item)
    if err != nil {
        t.Fatal(err)
    }

    var decoded *AmbiguousItem
    if err := ssz.Unmarshal(encoded, decoded); err != nil {
        t.Fatal(err)
    }

    if !reflect.DeepEqual(item, decoded) {
        t.Errorf("Expected %v, received %v", item, decodedjj)
    }
}
```

You'll see the following error message pop-up:

```
Expected { Field1: nil, Field2: 5 }, received { Field1: []. Field2: 5}
```

Our unmarshal prefers to unmarshal into an empty slice despite the original being nil, how can we resolve this? Typically, serialization libraries in Go run into similar roadblocks, and they suggest checking equality of objects within their scope by using their own custom comparison function. For this purpose, we had to write our own `ssz.DeepEqual` to catch these little nuances of SSZ which `reflect.DeepEqual` would not be able to discern.

No serialization specification is perfect in every language, but we can afford to make these compromises in Go for a better developer experience and to reduce ambiguity.

## Testing

Writing tests for a generic serialization is tricky, as they must be exhaustive, however, there are infinite types you can try your marshaling/unmarshaling rules on. How can we measure _robustness_? 

The standard approach for establishing robustness in Go is to include the following:

- Edge case unit tests
- Comprehensive fuzz tests

At a high level, the API for SSZ is quite simple and easy to understand. There are only 2 core functions needed for SSZ, which are `Marshal` and `Unmarshal`:

```go
func Marshal(val interface{}) ([]byte, error)

func Unmarshal(encoded []byte, val interface{}) error
```

Their conciseness is no accident, as we designed this API to match the very popular [JSON marshaler/unmarshaler](https://golang.org/pkg/encoding/json/#Marshal) from the Go standard library. Aside from hiding implementation details, the reason this was done was for powerful access to testing primitives in the form of _serialization fuzz testing_.

From [Techopedia](https://www.techopedia.com/definition/13625/fuzz-testing):

> Fuzz testing is a means of stress test applications by feeding random data into them in order to spot any errors or hang-ups that may occur. The idea behind fuzz testing is that software applications and systems can have a lot of different bugs or glitches related to data input.

Having fuzz tests can help us feed our round trip tests through all sorts of randomized data structures to ensure we catch those pesky panics and improve the robustness of our SSZ toolset.

For basic edge case, sanity-focused unit tests, we can opt for simple table-driven tests in Go to get the job done. We'll be doing a simple, round-trip marshal/unmarshal test to ensure we can recover data in its original form after passing through SSZ.

```go
type item struct {
    Field1 []byte
}

type nestedItem struct {
    Field1 []uint64
    Field2 *item
    Field3 [3]byte
}

var nestedItemExample = nestedItem{
    Field1: []uint64{2, 3, 4},
    Field2: &item{
        Field1: []byte{0},
    },
    Field3: [3]byte{1, 1, 1},
}

func TestMarshalUnmarshal(t *testing.T) {
    tests := []struct {
        input interface{}
        ptr   interface{}
    }{
        // Bool test cases.
        {input: true, ptr: new(bool)},
        {input: false, ptr: new(bool)},
        // Uint test cases.
        {input: byte(1), ptr: new(byte)},
        {input: uint16(232), ptr: new(uint16)},
        {input: uint32(1029391), ptr: new(uint32)},
        {input: uint64(23929309), ptr: new(uint64)},
        // Byte slice, byte array test cases.
        {input: [8]byte{1, 2, 3, 4, 5, 6, 7, 8}, ptr: new([8]byte)},
        {input: []byte{9, 8, 9, 8}, ptr: new([]byte)},
        // Basic type array test cases.
        {input: [12]uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}, ptr: new([12]uint64)},
        {input: [100]bool{true, false, true, true}, ptr: new([100]bool)},
        // Basic type slice test cases.
        {input: []bool{true, false, true, true, true}, ptr: new([]bool)},
        {input: []uint32{0, 0, 0}, ptr: new([]uint32)},
        // Struct decoding test cases.
        {input: nestedItemExample, ptr: new(nestedItem)},
        // Non-basic type slice/array test cases.
        {input: [2]nestedItem{nestedItemExample}, ptr: new([2]nestedItem)},
        // Pointer-type test cases.
        {input: &nestedItemExample, ptr: new(nestedItem)},
    }
    for _, tt := range tests {
        serializedItem, err := ssz.Marshal(tt.input)
        if err != nil {
            t.Fatal(err)
        }
        if err := ssz.Unmarshal(serializedItem, tt.ptr); err != nil {
            t.Fatal(err)
        }
        output := reflect.ValueOf(tt.ptr)
        inputVal := reflect.ValueOf(tt.input)
        if inputVal.Kind() == reflect.Ptr {
            if !ssz.DeepEqual(output.Interface(), tt.input) {
                t.Errorf("Expected %v, received %v", tt.input, output.Interface())
            }
        } else {
            got := output.Elem().Interface()
            want := tt.input
            if !ssz.DeepEqual(want, got) {
                t.Errorf("Did not unmarshal properly: wanted %v, received %v", tt.input, output.Elem().Interface())
            }
        }
    }
}
```

## Benchmarks and Standards

The final step for wrapping up our serialization library has to do with benchmarking and standardization. Note our focus of this implementation was a correct/robust _first_ approach without much attention to optimization. Using benchmarks and Go's pprof are the best ways to catch any relevant bottlenecks, but the most likely problem we will encounter is a massive amount of allocations per operation, as we are consistently creating new pointers, new reflect values, and more. 

Alec Thomas has put together an excellent repository standardizing serialization benchmarks in Go across different libraries you can find [here](https://github.com/alecthomas/go_serialization_benchmarks). We will be adding Go-SSZ to that list quite soon! In the meantime, expect a follow-up post on optimizing SSZ!

Check out our full repo here: [github.com/prysmaticlabs/go-ssz](github.com/prysmaticlabs/go-ssz)

