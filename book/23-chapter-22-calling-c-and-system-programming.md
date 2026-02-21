# Chapter 22: Calling C and System Programming

PHP extensions are written in C, but most PHP developers never touch them. Go makes C interop and system programming more accessible—though it's still advanced territory.

## CGO Basics

CGO allows Go to call C code:

```go
package main

/*
#include <stdio.h>
#include <stdlib.h>

void sayHello(const char* name) {
    printf("Hello, %s!\n", name);
}
*/
import "C"

import "unsafe"

func main() {
    name := C.CString("World")
    defer C.free(unsafe.Pointer(name))
    C.sayHello(name)
}
```

The comment before `import "C"` contains C code. CGO compiles it and generates bindings.

### C Types in Go

```go
// C types
var i C.int = 42
var f C.float = 3.14
var s *C.char = C.CString("hello")

// Converting to Go types
goInt := int(i)
goFloat := float32(f)
goString := C.GoString(s)
```

### Calling C Libraries

```go
/*
#cgo LDFLAGS: -lm
#include <math.h>
*/
import "C"

func main() {
    result := C.sqrt(16.0)
    fmt.Println(float64(result))  // 4.0
}
```

The `#cgo` directive sets compiler/linker flags.

### Memory Management

C memory must be managed manually:

```go
// Allocate
ptr := C.malloc(100)
defer C.free(ptr)

// String to C (allocates)
cstr := C.CString("hello")
defer C.free(unsafe.Pointer(cstr))

// C string to Go (copies)
gostr := C.GoString(cstr)
```

## When CGO Makes Sense

### Good Use Cases

1. **Wrapping existing C libraries**: SQLite, OpenSSL, image codecs
2. **Performance-critical code**: When pure Go isn't fast enough
3. **System interfaces**: Hardware access, kernel interfaces
4. **Legacy integration**: Existing C codebases

### When to Avoid CGO

1. **Cross-compilation**: CGO complicates builds
2. **Simple tasks**: Pure Go is often fast enough
3. **Concurrency**: CGO calls block OS threads
4. **Deployment**: Static binaries become harder

### CGO Trade-offs

| Pure Go | With CGO |
|---------|----------|
| Simple cross-compilation | Complex cross-compilation |
| Single static binary | May need shared libraries |
| Goroutine-friendly | Blocks OS threads |
| Memory-safe | Manual memory management |

## Syscalls and unsafe Package

For system programming without CGO:

```go
import (
    "syscall"
    "unsafe"
)

func getpid() int {
    pid, _, _ := syscall.Syscall(syscall.SYS_GETPID, 0, 0, 0)
    return int(pid)
}
```

### The unsafe Package

`unsafe` bypasses Go's type safety:

```go
import "unsafe"

// Get size
size := unsafe.Sizeof(int64(0))  // 8

// Pointer arithmetic
arr := [3]int{1, 2, 3}
ptr := unsafe.Pointer(&arr[0])
ptr2 := unsafe.Add(ptr, unsafe.Sizeof(int(0)))
val := *(*int)(ptr2)  // 2

// Type punning
var f float64 = 3.14
bits := *(*uint64)(unsafe.Pointer(&f))
```

### When to Use unsafe

- **Performance-critical code**: Avoiding copies
- **System interfaces**: Memory-mapped I/O
- **FFI**: Foreign function interface
- **Never in normal code**: There's usually a better way

## Building CLI Tools

Go excels at CLI tools—single binary, fast startup, cross-platform:

```go
package main

import (
    "flag"
    "fmt"
    "os"
)

func main() {
    verbose := flag.Bool("v", false, "verbose output")
    output := flag.String("o", "", "output file")
    flag.Parse()

    args := flag.Args()
    if len(args) == 0 {
        fmt.Fprintln(os.Stderr, "usage: mytool [-v] [-o file] input")
        os.Exit(1)
    }

    if *verbose {
        fmt.Println("Processing:", args[0])
    }

    // Process input
}
```

### Using Cobra for Complex CLIs

```go
import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
    Use:   "myapp",
    Short: "My application does amazing things",
}

var serveCmd = &cobra.Command{
    Use:   "serve",
    Short: "Start the server",
    Run: func(cmd *cobra.Command, args []string) {
        port, _ := cmd.Flags().GetInt("port")
        startServer(port)
    },
}

func init() {
    serveCmd.Flags().IntP("port", "p", 8080, "port to listen on")
    rootCmd.AddCommand(serveCmd)
}

func main() {
    rootCmd.Execute()
}
```

## Signal Handling

Handle OS signals for graceful shutdown:

```go
import (
    "os"
    "os/signal"
    "syscall"
)

func main() {
    // Setup
    server := startServer()

    // Handle signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    <-sigChan  // Block until signal received

    fmt.Println("Shutting down...")
    server.Shutdown(context.Background())
}
```

### Signal Handling Patterns

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())

    // Start workers with context
    go worker(ctx)

    // Handle signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    select {
    case sig := <-sigChan:
        log.Printf("Received signal: %v", sig)
        cancel()  // Cancel context, workers should stop
    }

    // Wait for workers (with timeout)
    time.Sleep(5 * time.Second)
}
```

### Handling SIGHUP for Config Reload

```go
func main() {
    config := loadConfig()

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGHUP)

    go func() {
        for range sigChan {
            log.Println("Reloading configuration...")
            config = loadConfig()
        }
    }()

    // Application runs with reloadable config
}
```

## Summary

- **CGO** enables calling C code but adds complexity
- **Use CGO** for existing C libraries or performance-critical code
- **unsafe** bypasses type safety—use sparingly
- **CLI tools** are a Go strength—single binary deployment
- **Signal handling** enables graceful shutdown and config reload

---

## Exercises

1. **CGO Hello World**: Write a Go program that calls a C function to print a message.

2. **Wrap a C Library**: Wrap a simple C library (e.g., math functions) with Go bindings.

3. **Memory Safety**: Demonstrate a memory leak in CGO code. Then fix it.

4. **CLI Tool**: Build a CLI tool with Cobra that has multiple subcommands and flags.

5. **Signal Handler**: Implement graceful shutdown on SIGINT with active request draining.

6. **SIGHUP Reload**: Implement configuration reload on SIGHUP without restart.

7. **unsafe Exploration**: Use unsafe to examine the memory layout of a struct.

8. **Cross-Compilation**: Cross-compile a Go program for multiple platforms. Then try with CGO.
