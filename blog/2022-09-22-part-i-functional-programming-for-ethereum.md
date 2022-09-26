---
layout: post
title:  "Blog Series: Pure, functional programming for modern blockchain development"
date: 2022-Sep-25
tags: 
  - functional-programming
---

Today, modern blockchains tend to be built in the popular Rust and Go programming languages for the 
myriad benefits they offer. In particular, Rust has risen to popularity with empowering developers to write
memory-safe software and great performance while Go has been a complentary choice for its powerful concurrency
and networking primitives while being easy to write. In domain of distributed systems engineering, writing 
software that is less bug-prone, more secure, and performant are what we tend to optimize for.

This blog series will focus on making the case for functional programming as a fantastic approach to blockchain development.
Each post will assume little-to-no familiarity with functional programming concepts, and focus on
showing how it can be used to solve _real problems_. We will not go into any mathematical topics and instead
only showcase how these concepts are useful by example. 

In each post, we'll introduce different concepts of functional programming and solve problems
we run into when designing distributed systems. We'll be using the Haskell programming language in this series and will be introducing 
language constructs as we go along. No need to be a Haskell expert to understand the premise of each post.
The series will culminate with guidance on how to develop software for production using a modern Haskell stack that actually works!

## What exactly is functional programming?

The most popular, modern languages today use imperative programming, which works as a series of defined procedures
to accomplish a task. In Go, for example, the `main()` function of your program can do anything as an ordered procedure of computation.

```go
func main() {
  db := openDB()
  recentBlocks := db.GetRecentBlocks()
  for _, block := range recentBlocks {
    if err := processBlock(block); err != nil {
      panic(err)
    }
    if err := saveBlock(block); err != nil {
      panic(err)
    }
  }
  if err := updateBlockOperationCaches(block); err != nil {
    panic(err)
  }
  ... // Do more operations...
}
```

In the example above, we need to read the program from top to bottom to fully understand what it must do. Each function makes sense
in isolation, mostly due to a descriptive name, but it is hard to reason about how they fit together until one 
reads this `main()` function in its entirety. **Maybe you didn't want to save any block if any block in the batch failed to process**.
This is harder to spot if you had a more complex codebase, and can sneak up on you without the compiler protecting you from the
bad intent of the code.

Functional programming (FP) flips this paradigm on its head. Instead, it follows a model of "declative programming" where we 
declare programs in terms of what things _are_ rather than procedures to be done. FP treats types, functions, and definitions 
as first-class citizens, and only allows deterministic, pure functions! In FP, all functions only depend on their input, 
cannot mutate data, and can only _transform_ their inputs into new values in a deterministic manner.

To illustrate, here's how we could produce a list of N `ones` in an imperative language (Go):

```go
func ones(n int) []int {
  result := make([]int, n)
  for i := 0; i < n; i++ {
    result[i] = 1
  }
  return result
}
```

Whereas in Haskell, we can an infinite list of `ones` in a simple way:

```haskell
ones :: [Integer] -- This is the type annotation. Ones is an (infinite) list of integers.
ones = 1:ones     -- The ":" is the list append operator in Haskell.
```

In Go, we have to define a procedure to get us a list of ones with a certain length, while in Haskell
we can define a recursive type that tells you exactly what `ones` is. The best part is this definition
is _lazy_ because we aren't actually computing an infinite list (which would blow up our computer). Instead,
we are merely defining what `ones` is, and then any other function could take any N numbers from this infinite
list as desired. We can do the same for prime numbers, natural numbers, or any other infinite sequence using
a functional, lazy definition.

In functional programming, function type signatures and their contents are enough to tell you what
exactly something is, sometimes in a visually striking manner. Here is how we would implement the popular
`quicksort` algorithm in this manner:

```haskell
quicksort :: (Ord a) => [a] -> [a] -- Type annotation of the quicksort function.
quicksort [] = []
quicksort (x:xs) =                -- Pattern matching a list with head element `x` and a tail `xs`
  let smaller = quicksort (filter (<=x) xs) in
  let bigger = quicksort (filter (>x) xs) in
  smaller ++ [x] ++ bigger
```

Instead of teling us a procedure for quicksorting with for-loops, we get a definition that visually depicts
what is going on in this recursive algorithm. 

`quicksort [4, 3, 5, 1, 2]` becomes

```
quicksort ([3, 1, 2]) ++ [4] ++ quicksort [5]
quicksort ([1, 2]) ++ [3] ++ [4] ++ quicksort [5]
quicksort [1] ++ [2] ++ [3] ++ [4] ++ quicksort [5]
[1] ++ [2] ++ [3] ++ [4] ++ [5]
[1, 2, 3, 4 5]
```

In functional programs, we define deterministic outputs in terms
of inputs. In the example above, quicksorting an empty list just returns an empty list.

## I won't switch to Haskell – how will this help me as an engineer?

Thinking functionally extends beyond Haskell and helps us brainstorm new ways of designing software that is easier to
maintain, test, and define. With fp, you can:

- Better manage your responsibilities when writing imperative software
- Understand how to "think with types", even in imperative code such as Go or Rust
- Learn cleaner abstractions that can better express the intent of your software

With fp, you can practice how to represent what you want your software to do in a more concise manner:

```haskell
let (validBlocks, invalidBlocks) = partition checkValidity incomingBlocks
```

## What's wrong with imperative programs?

Imperative programming is a pretty natural way of mapping a set of instructions to "get things done" in software.
Sometimes, procedures in imperative programs can be hundreds of lines of code, where we are accessing and updating databases, making
HTTP calls, interacting with end-users, and more. That kind of code can be hard to read and hard to maintain. On top of that, 
a lot of imperative code is neither deterministic nor pure, as we often introduce **side-effects** that are unintended in our programs.

Let's say we want to write an imperative program that writes down a procedure for putting together a sandwich. 
We might do something like this:

```go
sandwich.buns = prepareBuns()
if wantCheese() {
  sandwich.cheeses = inputCheeses()
} else {
  sandwich.cheeses = nil
}
sandwich.decideSauce = inputSauce()
return sandwich
```

In the code above, we mutate sandwich along the way, and nothing stops us from doing something crazy such as
making an HTTP request in the middle, or maybe sending an email when all we want is to build a sandwich!

### Thinking in types changes everything

Instead of saying how to build a sandwich, let's define what a sandwich _is_. In Haskell,
we "think in types" and build our entire definitions around type sets we have full control over.

```haskell
data Sandwich 
  = SquareBun Filling (Maybe Cheese) [Sauce] 
  | BurgerBun Filling (Maybe Cheese) [Sauce]
  | HotDogBun [Sauce]                        -- Yes hotdogs are sandwiches.

data Filling = Ham | Turkey | PeanutButter

data Cheese = American | Swiss | PepperJack | Cheddar 

data Sauce = Ketchup | Mustard | Mayo | Relish | Jelly
```

From here, instead of needing to define a _procedure_ to build a sandwich as a series of steps,
we simply define what a school lunch sandwich actually is in terms of its parts.

```haskell
schoolLunch :: Sandwich
schoolLunch = SquareBun PeanutButter Nothing [Jelly] -- No cheese because ew.
```

A school lunch above is simply a square-bun, pb&j sandwich! No need for an ordered mutation of sandwich data
when we can simply declare what we want.

### Technical debt

Functional programs tend to be comprised of many small functions that can be understood individually, whereas imperative programs
tend to grow into mega-sized functions and procedures that make refactoring a pain, and reasoning about the whole
program difficult. This is because **procedural order is key to imperative programs**. Imperative code is **easy to write but hard to read**.
When your code is being developed in a team environment, and a previous developer was the only one that understood all the logic
of a critical, 1000-line function, you might be in trouble. Modifying even a single line of a procedure could **cripple
a program or introduce a serious bug**, making it easier to grow technical debt.

### Side-effects

In procedural code, your operations can have unintended side-effects, because functions will be executed top-down as a series
of instructions without guarantees of determinism. In the middle of a for-loop, your code could access a database operation that
fails while you have already updated another database record before, leaving you in an inconsistent state. Because data is mutable,
your code can also suffer **data-races**, which are often patched up less-than-ideal tools.

**In pure, functional programs there can be no data races** because your data is **immutable** and each function is a
deterministic transformation from inputs to outputs. That is, side-effects are not allowed.

> No side-effects?! That seems impractical in the real world!

It sounds impossible to have a useful program that _does not_ have side-effects, as we use these all the time in
imperative programs to do valuable things. What alternative do we have in fp? It turns out we have a way of using
really cool techniques to kind of "emulate" imperative code using pure functions and still maintaining 
some awesome guarantees which we'll cover in this series.

In imperative code, however, your software could be doing something like this:

```go
// Simple addition of two integers.
func addNumbers(x, y int) {
  launch_missiles() // Here be dragons...
  return x + y
}
```

Although the example above is contrived, it is harder to stop bad side-effects when your functions are thousands of lines
long and have unclear naming. These kinds of dangerous side-effects _cannot_ happen in a pure functional language such 
as Haskell without making it obvious to the developer, because programs are transformations of data type declarations
rather than impure, ordered computation.

## Series overview

Over the course a blog series, we will be covering the major aspects of writing pure, functional code with real-world
examples and without any fancy math to go with it. We'll be introducing hard concepts in terms of what they can do for us
that other methods cannot, and wrap up the series with an analysis of how you can use this information especially if you
plan to continue working with imperative codebases (I am a Go developer myself!). The series outline is as follows:

- Thinking functionally
- Pure operations only! ...But what if I need side-effects?!
- Thinking in types and DSLs
- The cleanest code: interpretable programs
- Writing real programs in a functional language
- Why functional programming is not taking over the world yet

Stay tuned, as each post will be packed with content. I hope this peaks your interest as much as it has been a joy for me
to bring these ideas to developers that are not yet familiar with them.

