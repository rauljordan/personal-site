+++
title =  "Rust concepts I wish I learned earlier"
date = 2023-01-16

[taxonomies]
tags = ["rust"]

[extra]
photo = "https://user-images.githubusercontent.com/689247/27258102-0ddeb2ec-53a6-11e7-91f1-72b01ca3e4a0.png"
+++

![Image](https://user-images.githubusercontent.com/689247/27258102-0ddeb2ec-53a6-11e7-91f1-72b01ca3e4a0.png)

This past month, I have been enthralled by the [Rust programming language](<https://www.rust-lang.org/>)
given its unique edge for writing memory-safe, modern programs. Over the years, several languages have
emerged as the most preferred by engineers to write resilient, backend software. The tides have shifted
from Java/C++ into Go and Rust, which combine decades of programming language theory to build tools
that are effective in our current age.

Rust&rsquo;s numbers speak for themselves. As the [number 1 most loved language](<https://survey.stackoverflow.co/2022/?utm_source=so-owned&utm_medium=announcement-banner&utm_campaign=dev-survey-2022&utm_content=results#section-most-loved-dreaded-and-wanted-programming-scripting-and-markup-languages>) in
the famous stack overflow survey for **seven years** in a row, it has also recently been
released as part of Linux kernel - a feat no language other than C has been able to
accomplish. What&rsquo;s exciting about the language, to me, is that it provides something truly new
in the art of how software is built.

```rs
use std::thread;
use std::time::Duration;
use std::{collections::VecDeque, sync::Condvar, sync::Mutex};

fn main() {
    let queue = Mutex::new(VecDeque::new());

    thread::scope(|s| {
        let t = s.spawn(|| loop {
            let item = queue.lock().unwrap().pop_front();
            if let Some(item) = item {
                dbg!(item);
            } else {
                thread::park();
            }
        });

        for i in 0.. {
            queue.lock().unwrap().push_back(i);
            t.thread().unpark();
            thread::sleep(Duration::from_secs(1));
        }
    })
}
```

<!-- more -->

Rust, having picked up incredible usage in systems programming at large, also has a
reputation for being notoriously difficult to learn. Notwithstanding, there is a lot of
excellent Rust content catering to beginners and advanced
programmers alike. However, so many of them focus on the explaining the core mechanics of the
language and the concept of ownership rather than architecting applications.

As a Go developer writing highly concurrent programs and focusing on systems programming,
I hit a lot of bumps along the road in learning how to build real programs in Rust. That is,
if I were to port what I am currently working on into Rust, how effective would all those
tutorials be?

This blog post is meant to cover my experience going down the Rust rabbit hole and tips I wish
some learning resources could have taught better. Personally, I cannot learn a new language
from simply watching youtube videos, but rather through seeking out solutions for myself, making mistakes,
and feeling humbled by the process.

## On references

There are two kinds of references in Rust, a shared reference (also known as a borrow), and a
mutable reference (also known as an exclusive reference). Typically, these are seen as
`&x` and `&mut x` on a variable `x`. The difference between these two made a lot more sense once
I started calling the latter an &ldquo;exclusive reference&rdquo;.

Rust&rsquo;s reference model is fairly simple. Borrowers can have as many shared reference to something as
they need, but there can only be a single exclusive reference at a time. Otherwise, you could have
many callers trying to modify a value at the same time. If many borrowers could
also hold exclusive references, you risk undefined behavior, which safe Rust makes impossible.

Calling `&mut` &ldquo;exclusive&rdquo; references would have saved me some time while learning Rust.

```rs
pub struct Foo {
    x: u64,
}

impl Foo {
    /// Any type that borrows an instance of Foo can
    /// call this method, as it only requires a reference to Foo.
    pub fn total(&self) -> u64 {
        self.x
    }
    /// Only exclusive references to instances of Foo
    /// can call this method, as it requires Foo to be mutable.
    pub fn increase(&mut self) {
        self.x += 1;
    }
}

let foo = Foo { x: 10 };
println!("{}", foo.total()) // WORKS.
foo.increase() // ERROR: Foo not declared as mut
```

## Bidirectional references are possible

In other languages with garbage collection, it&rsquo;s easy to define graph data structures or
other types that contain references to some children, and those references could contain a reference
to their parent. In Rust, this is hard to do without fully understanding the borrowing rules. However, it
is still possible with methods provided by the standard library.

Let&rsquo;s say we have a struct called `Node` which contains a set of references to child nodes,
and also a reference to a parent node. Normally, Rust would complain, but we can satisfy the borrow checker
by wrapping the parent reference in something called a `Weak` pointer. This type tells Rust that
a node going away, or its children going away, shouldn&rsquo;t mean that its parent should also be dropped.

```rs
use std::cell::RefCell;
use std::rc::{Rc, Weak};

struct Node {
    value: i32,
    parent: RefCell<Weak<Node>>,
    children: RefCell<Vec<Rc<Node>>>,
}
```

This gives us handy primitives in building bidirectional references. However, I soon found out that building graph
data structures in Rust is *really* hard unless you know what you&rsquo;re doing, given the amount of book-keeping
one needs to do around modeling data effectively to satisfy the compiler.

## Implement Deref to make your code cleaner

Sometimes, we want to treat wrapper types as the things they contain. This is true for
common data structures such as `vec`, smart pointers such as `Box` or even the reference
counted types such as `Rc` and `Arc`. The standard lib contains traits called `Deref` and
`DerefMut` which will help you tell Rust how a type should be dereferenced.

```rs
use std::ops::{Deref, DerefMut};

struct Example<T> {
    value: T
}

impl<T> Deref for Example<T> {
    type Target = T;

    fn deref(&self) -> &Self::Target {
        &self.value
    }
}

impl<T> DerefMut for Example<T> {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.value
    }
}

let mut x = Example { value: 'a' };
*x = 'b';
assert_eq!('b', x.value);
```

In the example above, we can treat `*x` as if it were its underlying value of &ldquo;a&rdquo;, and even
mutate it because we defined rules for how it should dereference in either borrows or mutable
references. This is powerful and the reason why you don&rsquo;t need to worry about wrapping types
in smart pointers such as `Box`. The fact that a value is boxed is an implementation detail
which can be abstracted through these traits.

```rs
struct Foo {
    value: u64,
}
let mut foo = Box::new(Foo { value: 10 });

// Box implements DerefMut, so this will work fine!
*foo = Foo { value: 20 };
// Dot methods will work on foo because Box implements Deref.
// We do not have to worry about the implementation
// detail that Foo is boxed.
assert_eq!(20, foo.value);
```

## Be careful with methods on types that implement Deref

Ever wonder why methods such as `Arc::clone` exist when we could just do `.clone()` on
an Arc value? The reason has to do with how types implement Deref and is something
developers should be wary of. Consider the following example, where we are trying to implement
our own version of multi-producer/single-consumer (mpsc) channels from the standard library:

```rs
use std::sync::{Arc, Mutex, Condvar};

pub struct Sender<T> {
    inner: Arc<Inner<T>>,
}

impl<T> Sender<T> {
    pub fn send(&mut self, t: T) {
        ...
    }
}

impl<T: Clone> Clone for Sender<T> {
    fn clone(&self) -> Self {
        Self {
            // ERROR: Does not know whether to clone Arc or inner!
            inner: self.inner.clone(),
        }
    }
}

struct Inner<T> {
    ...
}

impl<T: Clone> Clone for Inner<T> {
    fn clone(&self) -> Self {
        ...
    }
}
```

In the example above, we have a `Sender` type we want to implement the `Clone` trait on.
This struct has a field called `inner` which is of type `Arc<Inner<T>>`. Recall that `Arc`
implements `Clone` already and *also* `Deref`. On top of that, our `Inner` also implements
`Clone`. With the code above, **Rust does not know** whether we want to clone Arc or the actual
inner value, so the code above will fail. In this case, we can use the actual method provided by Arc
from the sync crate.

```rs
impl<T: Clone> Clone for Sender<T> {
    fn clone(&self) -> Self {
        Self {
            // Now Rust knows to use the Clone method of Arc instead of the
            // clone method of inner itself.
            inner: Arc::clone(&self.inner),
        }
    }
}
```

## Understand when and when not to use interior mutability

Sometimes, you will need to use structures such as `Rc` or `Arc` in your code,
or implement structs that wrap some data and then want to mutate the data that is
being wrapped. Soon, you will hit a wall with the compiler telling you that
interior mutability is disallowed, which seems intractable at first sight.
However, there are ways of allowing interior mutability in Rust that are even
provided by the standard library.

One of the simplest is `Cell`, which gives you interior mutability of data. That is,
you could mutate the data within an `Rc` as long as the data is cheap to copy. You can achieve this
by wrapping your data within a `Rc<Cell<T>>`. It provides get and set methods, which do not even
need to be `mut`, because they copy data underneath the hood:

```rs
// impl<T: Copy> Cell<T>
pub fn get(&self) -> T

// impl<T> Cell<T>
pub fn set(&self, val: T)
```

Other types, such as `RefCell` help with moving certain borrow checks to runtime and skipping
some of the compiler&rsquo;s tough filters. However, this is risky, as it will panic at runtime
if borrow checks are not fulfilled. Treat the compiler as a friend and you shall be
rewarded. By skipping its checks or by pushing them to runtime, you are telling
the compiler &ldquo;trust me - what I am doing is sound&rdquo;

The std::cell package even warns us about this with a helpful passage:

    The more common inherited mutability, where one must have unique access to mutate a value, is one of the key language elements that enables Rust to reason strongly about pointer aliasing, statically preventing crash bugs. Because of that, inherited mutability is preferred, and interior mutability is something of a last resort. Since cell types enable mutation where it would otherwise be disallowed though, there are occasions when interior mutability might be appropriate, or even must be used, e.g.
    
    - Introducing mutability ‘inside’ of something immutable
    - Implementation details of logically-immutable methods.
    - Mutating implementations of Clone.


## Get and get mut methods are a thing

Many types, including `vec` implement both get and `get_mut` methods, letting you borrow and mutate
elements in the structure (the former only possible if you have a mutable reference to the collection).
It took me a while to know these options are available for many data structures and they helped make my life easier by 
writing clean code a lot easier!

```rs
let x = &mut [0, 1, 2];

if let Some(elem) = x.get_mut(1) {
    *elem = 42;
}
assert_eq!(x, &[0, 42, 2]);
```

## Embrace unsafe but sound code

As a Go developer, the &ldquo;unsafe&rdquo; package felt sacrilegeous and something I seldom touched. However, the notion
of unsafety in Rust is *very* different. In fact, a lot of the standard library uses &ldquo;unsafe&rdquo; to great
success! How is this possible? Although Rust&rsquo;s makes undefined behavior impossible, this does not apply to
code blocks that are marked as &ldquo;unsafe&rdquo;. Instead, a developer writing &ldquo;unsafe&rdquo; Rust
simply needs to guarantee its usage is sound to reap all the benefits.

Take the example below, where we have a function that returns the item at a specified
index in an array. To optimize this lookup, there is an unsafe function in Rust called
`get_unchecked` which is available on the array type. This will panic and lead to undefined behavior
if we attempt to get an index out of bounds. However, our function correctly asserts the unsafe call
will only happen if the index is less than the array length. This means the code below is
*sound* despite using an unsafe block.

```rs
/// Example taken from the Rustonomicon
fn item_at_index(idx: usize, arr: &[u8]) -> Option<u8> {
    if idx < arr.len() {
        unsafe {
            Some(*arr.get_unchecked(idx))
        }
    } else {
        None
    }
}
```

Embrace unsafe as long as you can prove the soundness of your API, but avoid exposing
functions that are directly unsafe to your consumers unless it is truly warranted. For this reason,
having tightly-controlled internals of your packages where you can prove unsafe blocks are sound
is a normal practice in Rust.

Typically, unsafe is used where performance is of absolute importance, or when you know of an easy
way to solve a problem using unsafe blocks *and* you can prove the soundness of your code.

## Use impl types as arguments rather than generic constraints when you can

Coming from Golang, I thought that traits could simply be provided as function
parameters all the time. For example:

```rs
trait Meower {
    fn meow(&self);
}

struct Cat {}

impl Meower for Cat {
    fn meow(&self) {
        println!("meow");
    }
}

// ERROR: Meower cannot be used as it does not have
// a size at compile time!
fn do_the_meow(meower: Meower) {
    meower.meow();
}
```

&#x2026;but the above fails, as trait objects do not have a size at compile time which
Rust needs in order to get the job done. We could get around it by adding `&dyn Meower`
and telling the compiler this is dynamically sized, but I soon learned this is not the &ldquo;rusty&rdquo;
solution. Instead, developers tend to pass in generic parameters *constrained* by a trait. For example:

```rs
fn do_the_meow<M: Meower>(meower: M) {
    meower.meow();
}
```

&#x2026;which compiles and passes. However, as functions get more complex, we might have a very hard-to-read
function signature if we also include other generic parameters. In this example, we don&rsquo;t really
need a generic type if all we want is to meow once. We don&rsquo;t even care about the results of the meow, so we
can instead rewrite as

```rs
fn do_the_meow(meower: &impl Meower) {
    meower.meow();
}
```

which tells the compiler &ldquo;I just want something that implements Meow&rdquo;. This pattern is a lot cleaner
when this is all you need, and there is no need for a generic return type of your function in the first place.

## iter() when you need to borrow, iter mut() for exclusive refs, and into iter() when you need to own

Many tutorials immediately jump to iterating over vectors using the `into_iter` method below:

```rs
let items = vec![1, 2, 3, 4, 5];
for item in items.into_iter() {
    println!("{}", item);
}
```

However, many beginners (myself included) hit a wall when we start using this iterator method
within structs, such as:

```rs
struct Foo {
    bar: Vec<u32>,
}

impl Foo {
    fn all_zeros(&self) -> bool {
        // ERROR: Cannot move out of self.bar!
        self.bar.into_iter().all(|x| x == 0)
    }
}
```

and immediately hit:

```rs
    error[E0507]: cannot move out of `self.bar` which is behind a shared reference
       --> src/main.rs:9:9
        |
    9   |         self.bar.into_iter().all(|x| x == 0)
        |         ^^^^^^^^ ----------- `self.bar` moved due to this method call
        |         |
        |         move occurs because `self.bar` has
        |         type `Vec<u32>`, which does not implement the `Copy` trait
```

After trying all kinds of approaches as a noob, I realized that `.into_iter()` takes *ownership*
of the collection, which is not what I needed for my purposes. Instead, there are two other useful methods
on iterators that I *wish* I had learned about earlier. The first is `.iter()`, which borrows the
collection, letting you assert things about its values but not own or mutate them, and also `.iter_mut()`
which helps you mutate internal values of the collection as long as you have the only exclusive reference.

In summary, use `.iter()` when you just need to borrow, `.into_iter()` when you want
take ownership, and `.iter_mut()` when you need to mutate elements of an iterator.

## Phantom data is more than just for working with raw pointers to types

Phantom data seems weird when you first encounter it, but it soon makes sense as a way
of telling the compiler one &ldquo;owns&rdquo; a certain value despite just having a raw pointer to it.
For example:

```rs
use std::marker;

struct Foo<'a, T: 'a> {
    bar: *const T,
    _marker: marker::PhantomData<&'a T>,
}
```

Tells the compiler that Foo *owns* T, despite only having a raw pointer to it. This is helpful
for applications that need to deal with raw pointers and use unsafe Rust. 

However, they can also be a way
to tell the compiler that your type does not implement the `Send` or `Sync` traits! You can
wrap the following types with PhantomData and use them in your structs as a way to tell the compiler
that your struct is neither Send nor Sync.

```rs
pub type PhantomUnsync = PhantomData<Cell<()>>;
pub type PhantomUnsend = PhantomData<MutexGuard<'static, ()>>;
```

## Use rayon for incremental parallelism

Sometimes, you want to parallelize work when iterating through collections, but hit a brick wall
when dealing with threading and making types safe to send across threads. Sometimes, the
extra boilerplate just isn&rsquo;t worth it if it makes your code almost unreadable.

Instead, there is an awesome package called [Rayon](<https://docs.rs/rayon/latest/rayon/iter/index.html>)
which provides fantastic tools for parallelizing your computations in a seamless manner. For example,
let&rsquo;s say we have a function that computes the sum of squares of an array.

```rs
fn sum_of_squares(input: &[i32]) -> i32 {
    input.iter()
            .map(|i| i * i)
            .sum()
}
```

The above can absolutely be parallelized due to the nature of multiplication and addition, and Rayon
makes it trivial to do so by giving us automatic access to &ldquo;parallel iterators&rdquo; on collections such
as arrays. Here&rsquo;s what it looks like with pretty much *zero* boilerplate. It also does not compromise
readability at all.

```rs
// Importing rayon prelude is what gives us access to .par_iter on arrays.
use rayon::prelude::*;

fn sum_of_squares(input: &[i32]) -> i32 {
    // We can use par_iter on our array to let rayon
    // handle the parallelization and reconciliation of
    // results at the end.
    input.par_iter()
            .map(|i| i * i)
            .sum()
}
```

## Understand the concept of extension traits when developing Rust libraries

So how does Rayon accomplish the above in such a clean way? The answer lies in &ldquo;extension traits&rdquo;,
which are traits that can be defined as extensions to other traits, such as `Iterator`. That is,
we can add other helpful functions to items that normally implement the `Iterator` trait,
but they will only be available if the trait is in scope, such as by importing it in a file.

This approach is excellent because these traits will only be available if you import the extension
trait in your project, and provide a great way to extend common collections and types with clean APIs
that developers can use just as easily as their normal counterparts. Using parallel iterators
is as easy as using iterators in Rust thanks to Rayon&rsquo;s extension traits.

In fact, there is a highly informative talk that explains how to use extension traits
to develop a library that provides progress bars on iterators [here](https://www.youtube.com/watch?v=bnnacleqg6k)

## Embrace the monadic nature of Option and Result types

After working with options and results, one will quickly see that \`.unwrap()\` moves
values out of them, which will fail if the option or result is part of a shared reference
such as a struct. However, sometimes all we want is to assert the option matches a value within
or to obtain a reference to its internals. There are many ways to do this, but one way
is to never leave the domain of options at all.

```rs
fn check_five(x: Option<i32>) -> bool {
    // Contains can just check if the Option has what we want.
    x.contains(&5)
}
```

Another example is one where we want to replace data inside of an option
with the None value, perhaps when interacting with some struct. We could write this
in an imperative programming manner, and do things verbosely as follows:

```rs
struct Foo {
    data: Option<T>,
}

impl<T> Foo<T> {
    // Takes the value of data and leaves None in its place.
    fn pop(&mut self) -> Option<T> {
        if self.data.is_none() {
            return None;
        }
        let value = self.data.unwrap();
        self.data = None;
        value
    }
}
```

However, Options have some really cool properties due to their fundamental nature
that they have useful methods defined on them which can make our lives a lot easier.

```rs
// Takes the value of data and leaves None in its place.
fn pop(&mut self) -> Option<T> {
    self.data.take()
}
```

Options in Rust are modeled after the same paradigm in functional programming languages, belonging
to a broader category of data types known as Monads. We won&rsquo;t into what those are, but just think
of them as wrappers around data that we can manipulate without needing to take things out of them. For example,
picture a function which adds the inner values of two options together and returns an option.

```rs
fn add(x: Option<i32>, y: Option<i32>) -> Option<i32> {
    if x.is_none() || y.is_none() {
        return None;
    }
    return Some(x.unwrap() + y.unwrap());
}
```

The above looks kind of clunky because of the none checks it needs to perform, and it also
sucks that we have to extract values out of both options and construct a *new* option out of that.
However, we can much better than this thanks to Option&rsquo;s special properties! Here&rsquo;s what we could do

```rs
fn add(x: Option<i32>, y: Option<i32>) -> Option<i32> {
    x.zip(y).map(|(a, b)| a+b)
}
```

We can zip and map options just like we can over arrays and vectors. This property is also found in
`Result` types, and even in things such as \`Future\` types. If you&rsquo;re curious about why this works,
learn more about Monads [here](<https://stackoverflow.com/questions/2704652/monad-in-plain-english-for-the-oop-programmer-with-no-fp-background>).

Embrace the monadic nature of the `Option` and `Result` types and don&rsquo;t just use
unwrap and if x.is<sub>none</sub>() {} else everywhere. They have so many useful methods defined which you can read about in
the standard library.

## Understand Drop how it should be implemented for different data structures

The standard library describes the Drop trait as:

When a value is no longer needed, Rust will run a “destructor” on that value. The most common way that a
value is no longer needed is when it goes out of scope.

```rs
pub trait Drop {
    fn drop(&mut self);
}
```

Drop is critical when writing data structures in Rust. One must have a sound approach towards how memory
will be thrown away once you no longer need it. Using reference-counted types can help you get over
these hurdles but it will not always be enough. For example, writing a custom linked list, or writing
structs that use channels, would typically need to implement a custom version of Drop. Implementing drop is actually
far easier than it seems, when you see how the standard lib actually does it:

```rs
// That's it!
fn drop<T>(t: T) {}
```

Using the clever rules of destruction upon losing scope, std::mem::drop has an empty function body! This is a
trick you can use in your own custom Drop implementations as long as you cover all of your bases.

## Really annoyed by the borrow checker? Use immutable data structures

Functional programmers love to say that global, mutable state is the root
of all evil, so why use it if you can avoid it? Thanks to Rust&rsquo;s functional constructs,
we are able to construct data structures that never need mutation the first place! This is especially
helpful when you need to write *pure* code similar to that seen in Haskell, OCaml, or other
languages.

With an example taken from a [comprehensive tutorial](<https://rust-unofficial.github.io/too-many-lists/third-final.html>) on linked lists, we see how one could build an immutable list where nodes are reference counted:

```rs
use std::rc::Rc;

pub struct List<T> {
    head: Link<T>,
}

type Link<T> = Option<Rc<Node<T>>>;

struct Node<T> {
    elem: T,
    next: Link<T>,
}

impl<T> List<T> {
    pub fn new() -> Self {
        List { head: None }
    }

    pub fn prepend(&self, elem: T) -> List<T> {
        List { head: Some(Rc::new(Node {
            elem: elem,
            next: self.head.clone(),
        }))}
    }

    pub fn tail(&self) -> List<T> {
        List { head: self.head.as_ref().and_then(|node| node.next.clone()) }
    }
    ...
```

This is awesome because it acts similarly to functional data structures where one does not
*modify* a list by prepending, but rather creates a list by constructing with the new element
as its head and the existing list as the tail.

```rs
    [head] ++ tail
```

Note that none of the methods above need to be `mut` because our data structure is immutable! This is also
efficient on memory because the structure is reference counted, meaning we won&rsquo;t be wasting
unnecessary resources duplicating the underlying memory of nodes if there are multiple callers on this
data structure.

Pure, functional code in Rust is neat, but many times, one will need tail recursion to write
code that is performant in this manner. However, be careful, as **tail-call optimization is not guaranteed
by the Rust compiler**

<https://stackoverflow.com/questions/59257543/when-is-tail-recursion-guaranteed-in-rust>

## Blanket traits help reduce duplication

Sometimes, you might want to constrain a generic parameter by many different traits:

```rs
struct Foo<T: Copy + Clone + Ord + Bar + Baz + Nyan> {
    vals: Vec<T>,
}
```

However, this can quickly get out of hand as soon as you start writing `impl`
statements, or when having multiple generic params. Instead, you can define a blanket
trait that can make your code a more DRY.

```rs
trait Fooer: Copy + Clone + Ord + Bar + Baz + Nyan {}

struct Foo<F: Fooer> {
    vals: Vec<F>,
}

impl<F: Fooer> Foo<F> { ... }
```

Blanket traits can help reduce duplication, however, don&rsquo;t let them get way too big. In many cases,
having a type require so many constraints might be a code-smell, as you are creating too large of
an abstraction. Instead, pass in concrete types if you notice your constraints get too large for no reason.
Certain applications, however, might benefit from blanket traits, such as libraries that aim
to provide as generic of an API as possible.

## Match statements are very flexible and structural in nature

Instead of nesting match statements, for example, one could bring values together as tuples and do
the following:

```rs
fn player_outcome(player: &Move, opp: &Move) -> Outcome {
    use Move::*;
    use Outcome::*;
    match (player, opp) {
        // Rock moves.
        (Rock, Rock) => Draw,
        (Rock, Paper) => Lose,
        (Rock, Scissors) => Win,
        // Paper moves.
        (Paper, Rock) => Win,
        (Paper, Paper) => Draw,
        (Paper, Scissors) => Lose,
        // Scissor moves.
        (Scissors, Rock) => Lose,
        (Scissors, Paper) => Win,
        (Scissors, Scissors) => Draw,
    }
}
```

This is an example of why pattern matching is far more powerful than switch statements seen in imperative
languages and it can do more much than that when it comes to deconstructing inner values!

## Avoid \_ => clauses in match statements if your matchees are finite and known

For example, let&rsquo;s say we have an enum:

```rs
enum Foo {
    Bar,
    Baz,
    Nyan,
    Zab,
    Azb,
    Bza,
}
```

When writing match statements, we should match every single type of the enum if we can and not
resort to catch-all clauses

```rs
match f {
    Bar => { ... },
    Baz => { ... },
    Nyan => { ... },
    Zab => { ... },
    Azb => { ... },
    Bza => { ... },
}
```

This is really helpful for maintenance of code, because if the original writer of the enum adds
more variants to it, the project won&rsquo;t compile if we forget to handle the new variants in our match statements.

## Match guard clauses are powerful

Match guards are awesome when you have an unknown or potentially infinite number of matchees, such as
ranges of numbers. However, they will force you to have a catch-all \`\_ =>\` if your range cannot be fully encompassed
by the guard, which can be a downside when writing maintainable code.

The canonical example from the [Rust book](<https://doc.rust-lang.org/rust-by-example/flow_control/match/guard.html>) is below:

```rs
enum Temperature {
    Celsius(i32),
    Fahrenheit(i32),
}

fn main() {
    let temperature = Temperature::Celsius(35);
    match temperature {
        Temperature::Celsius(t) if t > 30 => println!("{}C is above 30 Celsius", t),
        Temperature::Celsius(t) => println!("{}C is below 30 Celsius", t),
        Temperature::Fahrenheit(t) if t > 86 => println!("{}F is above 86 Fahrenheit", t),
        Temperature::Fahrenheit(t) => println!("{}F is below 86 Fahrenheit", t),
    }
}
```

## Need to mess with raw asm? There&rsquo;s a macro for that!

Core asm provides a macro that lets you write inline assembly in Rust, which can help when doing fancy
things such as directly intercepting the CPU&rsquo;s stack, or wanting to implement advanced optimizations.
Here&rsquo;s an example where we use inline assembly to trick the processor&rsquo;s stack to execute our
function by simply moving the stack pointer to it!

```rs
use core::arch::asm;

const MAX_DEPTH: isize = 48;
const STACK_SIZE: usize = 1024 * 1024 * 2;

#[derive(Debug, Default)]
#[repr(C)]
struct StackContext {
    rsp: u64,
}

fn nyan() -> ! {
    println!("nyan nyan nyan");
    loop {}
}

pub fn move_to_nyan() {
    let mut ctx = StackContext::default();
    let mut stack = vec![0u8; MAX as usize];
    unsafe {
        let stack_bottom = stack.as_mut_ptr().offset(MAX_DEPTH);
        let aligned = (stack_bottom as usize & !15) as *mut u8;
        std::ptr::write(aligned.offset(-16) as *mut u64, nyan as u64);
        ctx.rsp = aligned.offset(-16) as u64;
        switch_stack_to_fn(&mut ctx);
    }
}

unsafe fn switch_stack_to_fn(new: *const StackContext) {
    asm!(
        "mov rsp, [{0} + 0x00]",
        "ret",
        in(reg) new,
    )
}
```

## Use Criterion to benchmark your code and its throughput

The [Criterion](<https://bheisler.github.io/criterion.rs/book/criterion_rs.html>) package for benchmarking Rust code is a fantastic work of engineering. It gives you access to awesome benchmarking features with graphs,
regression analysis, and other fancy tools. It can even be used to measure different dimensions of your
functions such as time *and* throughput. For example, we can see how fast we can construct, take, and collect
raw bytes using the standard library&rsquo;s iterator methods at different histogram buckets.

```rs
use std::iter;

use criterion::BenchmarkId;
use criterion::Criterion;
use criterion::Throughput;
use criterion::{criterion_group, criterion_main};

fn from_elem(c: &mut Criterion) {
    static KB: usize = 1024;

    let mut group = c.benchmark_group("from_elem");
    for size in [KB, 2 * KB, 4 * KB, 8 * KB, 16 * KB].iter() {
        group.throughput(Throughput::Bytes(*size as u64));
        group.bench_with_input(BenchmarkId::from_parameter(size), size, |b, &size| {
            b.iter(|| iter::repeat(0u8).take(size).collect::<Vec<_>>());
        });
    }
    group.finish();
}

criterion_group!(benches, from_elem);
criterion_main!(benches);
```

and after adding the following entries to the project&rsquo;s Cargo.toml file, one can run it with \`cargo bench\`.

```toml
[dev-dependencies]
criterion = "0.3"

[[bench]]
name = "BENCH_NAME"
harness = false
```

Not only does criterion show you really awesome charts and descriptive info, but it
can also *remember* prior results of benchmark runs, telling you of performance regressions.
In this case, I was using my computer to do a lot of other things at the same time as I ran the benchmark, so it
naturally regressed from the last time I measured it. Nonetheless, this is really cool!

```bash
    Found 11 outliers among 100 measurements (11.00%)
      2 (2.00%) low mild
      4 (4.00%) high mild
      5 (5.00%) high severe
    from_elem/8192          time:   [79.816 ns 79.866 ns 79.913 ns]
                            thrpt:  [95.471 GiB/s 95.528 GiB/s 95.587 GiB/s]
                     change:
                            time:   [+7.3168% +7.9223% +8.4362%] (p = 0.00 < 0.05)
                            thrpt:  [-7.7799% -7.3407% -6.8180%]
                            Performance has regressed.
    Found 3 outliers among 100 measurements (3.00%)
      2 (2.00%) high mild
      1 (1.00%) high severe
    from_elem/16384         time:   [107.22 ns 107.28 ns 107.34 ns]
                            thrpt:  [142.15 GiB/s 142.23 GiB/s 142.31 GiB/s]
                     change:
                            time:   [+3.1408% +3.4311% +3.7094%] (p = 0.00 < 0.05)
                            thrpt:  [-3.5767% -3.3173% -3.0451%]
                            Performance has regressed.
```

## Understand key concepts by reading the std lib!

I love to get lost in some of the [standard library](<https://doc.rust-lang.org/std/>), especially std::rc, std::iter, and
std::collections. Here are some awesome tidbits I learned from it on my own:

-   How vec is actually implemented
-   The ways in which interior mutability is achieved by different methods in std::cell and std::rc
-   How channels are implemented in std::sync
-   The magic of std::sync::Arc
-   Hearing the thorough explanations of design decisions made while developing its libraries from Rust&rsquo;s authors

I hope this post was informative for folks coming into Rust and hitting some of its obstacles. Expect more Rust
content to come soon, especially on more advanced topics!

## Shoutout

Shoutout to my colleagues at Offchain Labs, Rachel and Lee Bousfield for their incredible breadth of knowledge of the language. Some of their tips inspired this post
