+++
title =  "Why Go's Error Handling is Awesome"
date = 2020-07-06

[taxonomies]
tags = ["golang"]
+++

Go's [infamous error handling](https://github.com/golang/go/issues/32825) has caught quite the attention from outsiders to the programming language, often touted as one of the language's most questionable design decisions. If you look into any project on Github written in Go, it's almost a guarantee you'll see the lines more frequently than anything else in the codebase:

```go
if err != nil {
    return err
}
```
Although it may seem redundant and unnecessary for those new to the language, the reason errors in Go are treated as first-class citizens (values) has a deeply-rooted history in programming language theory and the main goal of Go as a language itself. Numerous efforts have been made to change or improve how Go deals with errors, but so far, one proposal is winning above all others:

[Leave if err != nil alone!](https://github.com/golang/go/issues/32825)

<!-- more -->

## Go's error philosophy

Go's philosophy regarding error handling forces developers to incorporate errors as first class citizens of most functions they write. Even if you ignore an error using something like:

```go
func getUserFromDB() (*User, error) { ... }

func main() {
    user, _ := getUserFromDB()
}
```
Most linters or IDEs will catch that you're ignoring an error, and it will certaintly be visible to your teammates during code review. However, in other languages, it may not be clear that your code is not handling a potential exception in a `try catch` code block, being completely opaque about handling your control flow.

If you handle errors in Go the standard way, you get the benefits of:

1. No hidden control-flows
2. No unexpected `uncaught exception` logs blowing up your terminal (aside from actual program crashes via panics)
3. full-control of errors in your code as _values_ you can handle, return, and do anything you want with

Not only is the syntax of `func f() (value, error)` easy to teach to a newcomer, but also a standard in any Go project which ensures consistency. 

It's important to note Go's error syntax does not **force** you to handle every error your program may throw. Go simply provides a pattern to ensure you think of errors as critical to your program flow, but not much else. At the end of your program, if an error occurs, and you find it using `err != nil`, and your application doesn't do something actionable about it, you're in trouble either way - **Go can't save you**. Let's take a look at an example:

```go
if err := criticalDatabaseOperation(); err != nil {
    // Only logging the error without returning it to stop control flow (bad!)
    log.Printf("Something went wrong in the DB: %v", err)
    // WE SHOULD `return` beneath this line!
}

if err := saveUser(user); err != nil {
    return fmt.Errorf("Could not save user: %w", err)
}
```

If something goes wrong and `err != nil` in calling `criticalDatabaseOperation()`, we're not doing anything with the error aside from logging it! We might have data corruption or an otherwise unexpected issue that we are not handling intelligently, either via retrying the function call, canceling further program flow, or in worst-case scenario, shutting down the program. Go isn't magical and can't save you from these situations. Go only provides a standard approach for returning and using errors as values, but you still have to figure out how to handle the errors yourself.

### How other languages do it: throwing exceptions

In something like the Javascript Node.js runtime, you can structure your programs as follows, known as throwing `exceptions`:

```js
try {
    criticalOperation1();
    criticalOperation2();
    criticalOperation3();
} catch (e) {
    console.error(e);
}
```

If an error occurs in any of these functions, the stack trace for the error will pop up at runtime and will be logged to the console, but there is no explicit, programmatic handling of what went wrong.

Your `criticalOperation` functions don't need to explicitly handle error flow, as any exception that occurs within that try block will be raised at runtime along with a stack trace of what went wrong.
A benefit to exception-based languages is that, compared to Go, even an unhandled exception will still be raised via a stack trace at runtime if it occurs. In Go, it is possible to not handle a critical error at all, which can arguably be much worse. Go offers you full control of error handling, but also **full responsibility**.

EDIT: Exceptions are definitely not the only way other languages deal with errors. Rust,  for example, has a good compromise of using option types and pattern matching to find error conditions, leveraging some nice syntactic sugar to achieve similar results.

### Why Go doesn't use exceptions for error handling

#### The Zen of Go

The Zen of Go mentions two important proverbs:

1. Simplicity matters
2. Plan for failure, not success

Using the simple `if err != nil` snippet to all functions which return `(value, error)` helps ensure failure in your programs is thought of _first and foremost_. You don't need to wrangle with complicated, nested `try catch` blocks which appropriately handle all possible exceptions being raised.

#### Exception-based code can often be opaque

With exception-based code, however, you're forced to be aware of every situation in which your code could have exceptions without actually handling them, as they'll be caught by your `try catch` blocks. That is, it encourages programmers to never check errors, knowing that at the very least, some exception will be handled automatically at runtime if it occurs.

A function written in an exception-based programming language may often look like this:

```js
item = getFromDB()
item.Value = 400
saveToDB(item)
item.Text = 'price changed'
```

This code does nothing to ensure exceptions are properly handled. Perhaps the difference between making the code above become aware of exceptions is to switch the order of `saveToDB(item)` and `item.Text = 'price changed`, which is **opaque**, hard to reason about, and can encourage some lazy programming habits. In functional programming jargon, this is known as the fancy term: [violating referential transparency](https://stackoverflow.com/questions/28992625/exceptions-and-referential-transparency/28993780#28993780). This [blog post](https://devblogs.microsoft.com/oldnewthing/?p=36693) from Microsoft's engineering blog in 2005 still holds true today, namely:

> My point isn’t that exceptions are bad.
My point is that exceptions are too hard and I’m not smart
enough to handle them.

## Benefits of Go's error syntax

### Easy creation of actionable error chains

A superpower of the pattern `if err != nil` is how it allows for easy error-chains to traverse a program's hierarchy all the way to where they need to be handled. For example, a common Go error handled by a program's `main` function might read as follows:


[2020-07-05-9:00] ERROR: Could not create user: could not check if user already exists in DB: could not establish database connection: no internet

The error above is (a) clear, (b) actionable, (c) has sufficient context as to what layers of the application went _wrong_. Instead of blowing up with an unreadable, cryptic stack trace, errors like these that are a result of factors we can add human-readable context to, and should be handled via clear error chains as shown above.

Moreover, this type of error chain arises naturally as part of a standard Go program's structure, likely looking like this:

```go
// In controllers/user.go
if err := db.CreateUser(user); err != nil {
    return fmt.Errorf("could not create user: %w", err)
}

// In database/user.go
func (db *Database) CreateUser(user *User) error {
    ok, err := db.DoesUserExist(user)
    if err != nil {
        return fmt.Errorf("could not check if user already exists in db: %w", err)
    }
    ...
}

func (db *Database) DoesUserExist(user *User) error {
    if err := db.Connected(); err != nil {
        return fmt.Errorf("could not establish db connection: %w", err)
    }
    ...
}

func (db *Database) Connected() error {
    if !hasInternetConnection() {
        return errors.New("no internet connection")
    }
    ...
}
```

The beauty of the code above is that each of these errors are completely namespaced by their respective function, are informative, and only handle responsibility for what they are aware of. This sort of error chaining using `fmt.Errorf("something went wrong: %w", err)` makes it trivial to build awesome error messages that can tell you _exactly_ what went wrong based on how _you_ defined it. 

On top of this, if you want to also attach a stack trace to your functions, you can utilize the fantastic [github.com/pkg/errors](https://godoc.org/github.com/pkg/errors) library, giving you functions such as:

```go
errors.Wrapf(err, "could not save user with email %s", email)
```

which print out a stack trace _along_ with the human-readable error chain you created through your code. If I could summarize the most important pieces of advice I've received regarding writing idiomatic error handling in Go:

1. **Add stack traces when your errors are actionable to developers**

2. **Do something with your returned errors, don't just bubble them up to main, log them, and forget them**

3. **Keep your error chains unambiguous**

When I write Go code, error handling is the one thing I _never_ worry about, because errors themselves are a central aspect of every function I write, giving me full control in how I handle them safely, in a readable manner, and responsibly.

> "if ...; err != nil" is something you'll probably type if you write go. I don't think it's a plus or a negative. It gets the job done, it's easy to understand, and it empowers the programmer to do the right thing when the program fails. The rest is up to you.

\- From [Hacker News](https://news.ycombinator.com/item?id=20303468)

## Key Readings 

- [Leave if err != nil alone!](https://github.com/golang/go/issues/32825)
- [Go pkg errors](https://godoc.org/github.com/pkg/errors)
