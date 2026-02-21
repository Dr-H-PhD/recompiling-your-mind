# Appendix G: Glossary

A comprehensive glossary of Go terms with PHP equivalents where applicable.

---

**Blank Identifier (`_`)**
: A special identifier that discards values. Used when a function returns multiple values but you only need some of them.
: *PHP equivalent:* `list($a, , $c) = $array;` (skipping values)

**Buffered Channel**
: A channel with capacity to hold values without a ready receiver. Created with `make(chan T, size)`.
: *PHP equivalent:* None. Similar concept to a queue with limited capacity.

**Channel**
: A typed conduit for communication between goroutines. Enables safe concurrent programming.
: *PHP equivalent:* None. Closest analogue is Symfony Messenger transport.

**Closure**
: An anonymous function that captures variables from its enclosing scope.
: *PHP equivalent:* Anonymous functions with `use` keyword: `function() use ($var) { ... }`

**Composition**
: Building complex types by combining simpler types through embedding, rather than inheritance.
: *PHP equivalent:* Using traits or dependency injection instead of class inheritance.

**Context**
: A package (`context`) providing request-scoped values, cancellation signals, and deadlines across API boundaries.
: *PHP equivalent:* Request attributes in Symfony; no equivalent for cancellation.

**Defer**
: A statement that schedules a function call to run when the enclosing function returns.
: *PHP equivalent:* `finally` blocks in try-catch; register_shutdown_function() loosely.

**Embedding**
: Including one struct type within another to promote its fields and methods.
: *PHP equivalent:* Traits, but embedding is more like composition than inheritance.

**Error**
: A built-in interface type representing error conditions. Go's primary error handling mechanism.
: *PHP equivalent:* Exceptions, but errors are values, not thrown.

**Exported**
: Identifiers starting with uppercase letters are exported (public). Lowercase are unexported (private).
: *PHP equivalent:* `public` vs `private` keywords.

**fmt**
: Standard library package for formatted I/O. Used for printing and string formatting.
: *PHP equivalent:* `printf()`, `sprintf()`, `echo`.

**Goroutine**
: A lightweight thread of execution managed by Go's runtime. Starts with `go` keyword.
: *PHP equivalent:* None directly. Similar to threads but much lighter weight.

**GOPATH**
: The location of your Go workspace (mostly obsolete with Go modules).
: *PHP equivalent:* Composer's vendor directory concept, loosely.

**Go Modules**
: The dependency management system using `go.mod` and `go.sum` files.
: *PHP equivalent:* Composer with `composer.json` and `composer.lock`.

**HTTP Handler**
: An interface with `ServeHTTP(ResponseWriter, *Request)` method for handling HTTP requests.
: *PHP equivalent:* Controller action in Symfony; PSR-15 handler.

**Interface**
: A set of method signatures. Types implicitly satisfy interfaces by implementing the methods.
: *PHP equivalent:* Interfaces, but Go interfaces are satisfied implicitly (no `implements`).

**Internal Package**
: Packages in an `internal/` directory are only importable by code in the parent tree.
: *PHP equivalent:* Convention only; no enforced equivalent in PHP.

**Make**
: Built-in function to create slices, maps, and channels.
: *PHP equivalent:* Array creation: `[]`, `array()`.

**Map**
: A built-in associative data structure with key-value pairs.
: *PHP equivalent:* Associative arrays: `['key' => 'value']`.

**Method**
: A function with a receiver argument, called on a specific type.
: *PHP equivalent:* Class methods: `public function name() { ... }`

**Method Set**
: The set of methods associated with a type. Determines which interfaces the type satisfies.
: *PHP equivalent:* All public methods of a class.

**Module**
: A collection of Go packages with a `go.mod` file defining the module path and dependencies.
: *PHP equivalent:* A Composer package.

**Mutex**
: A mutual exclusion lock for protecting shared data from concurrent access.
: *PHP equivalent:* flock() for file locking; Redis locks for distributed systems.

**New**
: Built-in function that allocates memory and returns a pointer to the zero value.
: *PHP equivalent:* `new ClassName()`, but returns pointer.

**Nil**
: The zero value for pointers, interfaces, maps, slices, channels, and function types.
: *PHP equivalent:* `null`.

**Package**
: A directory of Go source files with a package declaration. The unit of compilation.
: *PHP equivalent:* A namespace combined with a directory.

**Panic**
: A built-in function that stops ordinary flow and begins panicking. Should be rare.
: *PHP equivalent:* `throw new Exception()`, but more severe.

**Pointer**
: A value that holds the memory address of another value.
: *PHP equivalent:* References with `&`, but more explicit in Go.

**Range**
: Keyword for iterating over arrays, slices, strings, maps, and channels.
: *PHP equivalent:* `foreach`.

**Receiver**
: The value or pointer on which a method is called.
: *PHP equivalent:* `$this` in a class method.

**Recover**
: Built-in function to regain control after a panic.
: *PHP equivalent:* `catch` block in try-catch.

**Rune**
: An alias for `int32`, representing a Unicode code point.
: *PHP equivalent:* `mb_ord()` return value; a single Unicode character.

**Select**
: A control structure for waiting on multiple channel operations.
: *PHP equivalent:* None. Conceptually similar to `stream_select()`.

**Slice**
: A dynamically-sized view into an underlying array.
: *PHP equivalent:* Arrays, but slices have capacity and length concepts.

**Struct**
: A composite type grouping together variables under a single type.
: *PHP equivalent:* A class with only public properties.

**Struct Tag**
: Metadata attached to struct fields, commonly used for JSON encoding.
: *PHP equivalent:* Doctrine annotations; PHP 8 attributes.

**Type Assertion**
: Extracting the concrete type from an interface value.
: *PHP equivalent:* `instanceof` check followed by cast.

**Type Switch**
: A switch statement that compares types rather than values.
: *PHP equivalent:* Multiple `instanceof` checks.

**Unbuffered Channel**
: A channel with no capacity. Send blocks until receive, and vice versa.
: *PHP equivalent:* None. Synchronous handoff between goroutines.

**Variadic Function**
: A function that accepts a variable number of arguments.
: *PHP equivalent:* `func_get_args()` or `...$args` syntax.

**WaitGroup**
: A synchronisation primitive for waiting for a collection of goroutines to finish.
: *PHP equivalent:* None directly. Similar to Promise.all() in JavaScript.

**Zero Value**
: The default value for a type when not explicitly initialised.
: *PHP equivalent:* No direct equivalent. Variables must be initialised in PHP.

---

## Common Abbreviations

| Abbreviation | Meaning |
|--------------|---------|
| API | Application Programming Interface |
| CGO | C Go (Go's C interoperability) |
| CLI | Command Line Interface |
| CORS | Cross-Origin Resource Sharing |
| CPU | Central Processing Unit |
| DI | Dependency Injection |
| DSL | Domain-Specific Language |
| DTO | Data Transfer Object |
| FIFO | First In, First Out |
| GC | Garbage Collector |
| GOROOT | Go installation directory |
| HTTP | Hypertext Transfer Protocol |
| I/O | Input/Output |
| JSON | JavaScript Object Notation |
| JWT | JSON Web Token |
| M | Machine (OS thread in scheduler) |
| MVC | Model-View-Controller |
| ORM | Object-Relational Mapping |
| P | Processor (logical processor in scheduler) |
| REST | Representational State Transfer |
| RPC | Remote Procedure Call |
| SQL | Structured Query Language |
| TLS | Transport Layer Security |
| URL | Uniform Resource Locator |
| UUID | Universally Unique Identifier |
