# Table of Contents

## Part I: The Mental Shift

1. **Why Your PHP Brain Fights Go**
   - The curse of expertise
   - Interpreted vs compiled
   - Dynamic vs static typing
   - "It just works" vs "prove it works"

2. **Philosophy Differences**
   - PHP: "Get it done, fix it later"
   - Go: "Do it right, do it once"
   - Explicit over implicit
   - Simplicity over expressiveness

3. **The Type System Transition**
   - From dynamic to static
   - Type inference as compromise
   - Generics and union types
   - Type assertions
   - Generics deep dive: constraints and patterns
   - Data structures: PHP/SPL vs Go
   - Memory model differences

4. **Error Handling—The Hardest Shift**
   - Why `if err != nil` feels wrong
   - Exceptions vs explicit errors
   - Error wrapping
   - Custom error types

## Part II: Structural Rewiring

5. **From Classes to Structs**
   - No constructors
   - Methods as functions with receivers
   - Value vs pointer receivers
   - Visibility via case

6. **Inheritance Is Dead—Long Live Composition**
   - Why Go has no inheritance
   - Embedding
   - Interface composition
   - Flattening hierarchies

7. **Interfaces—Go's Hidden Superpower**
   - Implicit satisfaction
   - Small interfaces
   - Accept interfaces, return structs
   - The empty interface

8. **Packages and Modules**
   - Explicit imports
   - `go.mod` vs `composer.json`
   - Internal packages
   - No circular imports

9. **The Standard Library Is Your Framework**
   - `net/http` vs HttpFoundation
   - `encoding/json` vs Serializer
   - `database/sql` vs Doctrine DBAL
   - `html/template` vs Twig

## Part III: Practical Patterns

10. **Web Development Without a Framework**
    - HTTP handlers
    - Middleware patterns
    - Routing
    - Request validation
    - HTTP clients and external APIs
    - Gin and Echo frameworks
    - WebSockets and real-time communication

11. **Database Access**
    - `database/sql` fundamentals
    - Query builders and ORMs
    - Migrations
    - Connection pooling
    - NoSQL: MongoDB, Redis
    - Data streaming: Kafka, Redis Streams

12. **API Development**
    - JSON encoding/decoding
    - OpenAPI integration
    - Authentication middleware
    - Validation patterns
    - gRPC: all streaming patterns, resilience, TLS/mTLS
    - GraphQL with gqlgen

13. **Testing—A Different Philosophy**
    - Table-driven tests
    - No assertions library
    - Mocking with interfaces
    - Benchmarking

14. **Configuration and Environment**
    - No `.env` magic
    - Viper patterns
    - 12-factor principles
    - Secret management

## Part IV: Concurrency—The New Frontier

15. **Introducing Concurrency**
    - What PHP doesn't have
    - Goroutines vs processes
    - The Go scheduler
    - Mental model shift

16. **Channels—Message Passing**
    - Typed channels
    - Buffered vs unbuffered
    - Channel directions
    - Range over channels

17. **Select and Coordination**
    - Select statements
    - Timeouts and deadlines
    - Context package
    - WaitGroups

18. **Concurrency Patterns**
    - Worker pools
    - Fan-out/fan-in
    - Pipeline processing
    - Graceful shutdown

19. **When Concurrency Goes Wrong**
    - Race conditions
    - The race detector
    - Deadlocks
    - Channel leaks

## Part V: Advanced Topics

20. **Reflection and Code Generation**
    - reflect package
    - When to use reflection
    - `go generate`
    - Build-time vs runtime

21. **Performance Optimisation**
    - Profiling with pprof
    - Memory allocation patterns
    - Escape analysis
    - Pool patterns

22. **Calling C and System Programming**
    - CGO basics
    - Syscalls
    - CLI tools
    - Signal handling

## Part VI: Production Systems

23. **Building and Deploying**
    - Single binary deployment
    - Cross-compilation
    - Docker multi-stage builds
    - Systemd services
    - Kubernetes and Helm
    - Service mesh with Istio
    - GitOps with Argo CD

24. **Observability**
    - Structured logging
    - Prometheus metrics
    - OpenTelemetry tracing
    - Health checks

25. **Migration Strategies**
    - Strangler fig pattern
    - Side-by-side execution
    - API gateway approaches
    - Case study

26. **Security**
    - OWASP Top 10 in Go
    - Password hashing and encryption
    - TLS configuration
    - Secrets management
    - CORS and security headers

27. **Distributed Systems**
    - CAP theorem
    - Service discovery
    - Circuit breakers
    - Saga pattern
    - Leader election

28. **Building CLI Tools**
    - The flag package
    - Subcommands and argument parsing
    - User input and output
    - Progress bars and colours
    - Testing CLI applications

29. **File I/O**
    - io.Reader and io.Writer interfaces
    - Buffered I/O with bufio
    - Working with paths
    - JSON and CSV processing
    - Concurrent file processing

## Appendices

A. **PHP-to-Go Phrasebook**
B. **Standard Library Essentials**
C. **Common Pitfalls and Best Practices**
D. **Symfony-to-Go Service Mapping**
E. **Recommended Reading**
