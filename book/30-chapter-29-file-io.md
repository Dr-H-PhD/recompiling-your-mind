# Chapter 29: File I/O

PHP's file functions are convenient: `file_get_contents()`, `file_put_contents()`, `fopen()`/`fread()`. Go's approach is more explicit, built around the `io.Reader` and `io.Writer` interfaces that form the foundation of all I/O operations.

## Why File I/O Matters

File operations are essential for:

- **Configuration**: Reading YAML, JSON, or INI files
- **Logging**: Writing structured logs to files
- **Data Processing**: Parsing CSV, transforming data
- **Caching**: Storing computed results
- **Communication**: Named pipes, socket files

## PHP vs Go: Quick Comparison

| Operation | PHP | Go |
|-----------|-----|-----|
| Read entire file | `file_get_contents($path)` | `os.ReadFile(path)` |
| Write entire file | `file_put_contents($path, $data)` | `os.WriteFile(path, data, perm)` |
| Open file | `fopen($path, 'r')` | `os.Open(path)` |
| Read line by line | `fgets($handle)` | `bufio.Scanner` |
| File exists | `file_exists($path)` | `os.Stat(path)` |
| Create directory | `mkdir($path, 0755, true)` | `os.MkdirAll(path, 0755)` |

## Basic File Operations

### Reading Files

```go
// Read entire file into memory (like file_get_contents)
func readFile(path string) ([]byte, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("reading %s: %w", path, err)
    }
    return data, nil
}

// Read as string
func readFileAsString(path string) (string, error) {
    data, err := os.ReadFile(path)
    return string(data), err
}

// Usage
content, err := os.ReadFile("config.json")
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(content))
```

### Writing Files

```go
// Write entire file (like file_put_contents)
func writeFile(path string, data []byte) error {
    // 0644 = owner rw, group r, others r
    return os.WriteFile(path, data, 0644)
}

// Write string
func writeString(path, content string) error {
    return os.WriteFile(path, []byte(content), 0644)
}

// Usage
config := `{"port": 8080}`
if err := os.WriteFile("config.json", []byte(config), 0644); err != nil {
    log.Fatal(err)
}
```

### File Handles

For more control, use `os.Open()` and `os.Create()`:

```go
// Reading with file handle
func readWithHandle(path string) error {
    file, err := os.Open(path)  // Read-only
    if err != nil {
        return err
    }
    defer file.Close()  // Always close!

    data, err := io.ReadAll(file)
    if err != nil {
        return err
    }

    fmt.Println(string(data))
    return nil
}

// Writing with file handle
func writeWithHandle(path string, data []byte) error {
    file, err := os.Create(path)  // Create or truncate
    if err != nil {
        return err
    }
    defer file.Close()

    _, err = file.Write(data)
    return err
}

// Open with specific flags
func openForAppend(path string) (*os.File, error) {
    return os.OpenFile(path,
        os.O_APPEND|os.O_CREATE|os.O_WRONLY,
        0644)
}
```

## The io.Reader and io.Writer Interfaces

Go's I/O is built on two fundamental interfaces:

```go
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}
```

Files, network connections, HTTP bodies, and buffers all implement these interfaces. This enables composable I/O:

```go
// Copy from any Reader to any Writer
func processData(r io.Reader, w io.Writer) error {
    _, err := io.Copy(w, r)
    return err
}

// Works with files
src, _ := os.Open("input.txt")
dst, _ := os.Create("output.txt")
processData(src, dst)

// Works with HTTP responses
resp, _ := http.Get("https://example.com")
processData(resp.Body, os.Stdout)

// Works with strings
processData(strings.NewReader("hello"), os.Stdout)
```

### Common io Functions

```go
// Copy all data
written, err := io.Copy(dst, src)

// Copy with size limit
written, err := io.CopyN(dst, src, 1024)  // Max 1KB

// Read exactly n bytes
buf := make([]byte, 100)
_, err := io.ReadFull(reader, buf)

// Read until delimiter
data, err := io.ReadAll(reader)

// Combine multiple readers
combined := io.MultiReader(reader1, reader2, reader3)

// Write to multiple destinations
multi := io.MultiWriter(file, os.Stdout)
multi.Write([]byte("logged to file and console"))

// Limit reader size
limited := io.LimitReader(reader, 1024*1024)  // Max 1MB
```

## Buffered I/O

PHP buffers I/O automatically. Go requires explicit buffering for performance:

### Reading Line by Line

```go
import "bufio"

func readLines(path string) ([]string, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var lines []string
    scanner := bufio.NewScanner(file)

    for scanner.Scan() {
        lines = append(lines, scanner.Text())
    }

    return lines, scanner.Err()
}

// Process large files without loading into memory
func processLines(path string, fn func(string) error) error {
    file, err := os.Open(path)
    if err != nil {
        return err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    lineNum := 0

    for scanner.Scan() {
        lineNum++
        if err := fn(scanner.Text()); err != nil {
            return fmt.Errorf("line %d: %w", lineNum, err)
        }
    }

    return scanner.Err()
}

// Usage
err := processLines("data.csv", func(line string) error {
    fields := strings.Split(line, ",")
    // Process fields...
    return nil
})
```

### Custom Scanner Delimiters

```go
// Split on custom delimiter
scanner := bufio.NewScanner(file)
scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
    // Split on double newlines (paragraphs)
    if i := bytes.Index(data, []byte("\n\n")); i >= 0 {
        return i + 2, data[:i], nil
    }
    if atEOF && len(data) > 0 {
        return len(data), data, nil
    }
    return 0, nil, nil
})

for scanner.Scan() {
    paragraph := scanner.Text()
    fmt.Println("---")
    fmt.Println(paragraph)
}
```

### Buffered Writing

```go
func writeLines(path string, lines []string) error {
    file, err := os.Create(path)
    if err != nil {
        return err
    }
    defer file.Close()

    writer := bufio.NewWriter(file)

    for _, line := range lines {
        _, err := writer.WriteString(line + "\n")
        if err != nil {
            return err
        }
    }

    // Must flush buffer!
    return writer.Flush()
}

// Buffered writer with automatic flushing
func logWriter(path string) (*bufio.Writer, func() error) {
    file, _ := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    writer := bufio.NewWriter(file)

    cleanup := func() error {
        if err := writer.Flush(); err != nil {
            file.Close()
            return err
        }
        return file.Close()
    }

    return writer, cleanup
}
```

## Working with Paths

```go
import "path/filepath"

// Join paths (handles separators)
configPath := filepath.Join(home, ".config", "myapp", "config.json")

// Get directory and filename
dir := filepath.Dir("/path/to/file.txt")   // "/path/to"
file := filepath.Base("/path/to/file.txt") // "file.txt"
ext := filepath.Ext("/path/to/file.txt")   // ".txt"

// Absolute path
abs, err := filepath.Abs("./relative/path")

// Clean path (resolve . and ..)
clean := filepath.Clean("/path/to/../other")  // "/path/other"

// Match patterns
matched, err := filepath.Match("*.txt", "readme.txt")  // true

// Walk directory tree
filepath.Walk("/path/to/dir", func(path string, info os.FileInfo, err error) error {
    if err != nil {
        return err
    }
    if !info.IsDir() {
        fmt.Println(path)
    }
    return nil
})

// Glob patterns
files, err := filepath.Glob("*.go")
```

## File Information

```go
// Get file info
info, err := os.Stat(path)
if err != nil {
    if os.IsNotExist(err) {
        fmt.Println("file doesn't exist")
    }
    return err
}

fmt.Println("Name:", info.Name())
fmt.Println("Size:", info.Size())
fmt.Println("Mode:", info.Mode())
fmt.Println("ModTime:", info.ModTime())
fmt.Println("IsDir:", info.IsDir())

// Check if file exists
func fileExists(path string) bool {
    _, err := os.Stat(path)
    return err == nil
}

// Check if directory
func isDirectory(path string) bool {
    info, err := os.Stat(path)
    return err == nil && info.IsDir()
}
```

## Directory Operations

```go
// Create directory
err := os.Mkdir("newdir", 0755)

// Create nested directories (like mkdir -p)
err := os.MkdirAll("path/to/nested/dir", 0755)

// Remove file or empty directory
err := os.Remove("file.txt")

// Remove directory and contents
err := os.RemoveAll("directory")

// Rename/move
err := os.Rename("old.txt", "new.txt")

// Read directory contents
entries, err := os.ReadDir(".")
for _, entry := range entries {
    info, _ := entry.Info()
    fmt.Printf("%s %d\n", entry.Name(), info.Size())
}

// Create temp directory
tmpDir, err := os.MkdirTemp("", "myapp-*")
defer os.RemoveAll(tmpDir)

// Create temp file
tmpFile, err := os.CreateTemp("", "data-*.json")
defer os.Remove(tmpFile.Name())
```

## JSON File Operations

```go
// Read JSON file
func readJSON[T any](path string) (T, error) {
    var result T

    data, err := os.ReadFile(path)
    if err != nil {
        return result, err
    }

    err = json.Unmarshal(data, &result)
    return result, err
}

// Write JSON file (formatted)
func writeJSON(path string, v interface{}) error {
    data, err := json.MarshalIndent(v, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, 0644)
}

// Stream large JSON
func streamJSONArray(path string, fn func(json.RawMessage) error) error {
    file, err := os.Open(path)
    if err != nil {
        return err
    }
    defer file.Close()

    decoder := json.NewDecoder(file)

    // Read opening bracket
    _, err = decoder.Token()
    if err != nil {
        return err
    }

    // Read array elements
    for decoder.More() {
        var raw json.RawMessage
        if err := decoder.Decode(&raw); err != nil {
            return err
        }
        if err := fn(raw); err != nil {
            return err
        }
    }

    return nil
}
```

## CSV Processing

```go
import "encoding/csv"

// Read CSV
func readCSV(path string) ([][]string, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    reader := csv.NewReader(file)
    return reader.ReadAll()
}

// Stream large CSV
func processCSV(path string, fn func([]string) error) error {
    file, err := os.Open(path)
    if err != nil {
        return err
    }
    defer file.Close()

    reader := csv.NewReader(file)

    for {
        record, err := reader.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }
        if err := fn(record); err != nil {
            return err
        }
    }

    return nil
}

// Write CSV
func writeCSV(path string, records [][]string) error {
    file, err := os.Create(path)
    if err != nil {
        return err
    }
    defer file.Close()

    writer := csv.NewWriter(file)

    for _, record := range records {
        if err := writer.Write(record); err != nil {
            return err
        }
    }

    writer.Flush()
    return writer.Error()
}
```

## Concurrent File Processing

```go
// Process multiple files concurrently
func processFilesConcurrently(files []string, fn func(string) error) error {
    var wg sync.WaitGroup
    errCh := make(chan error, len(files))

    for _, file := range files {
        wg.Add(1)
        go func(f string) {
            defer wg.Done()
            if err := fn(f); err != nil {
                errCh <- fmt.Errorf("%s: %w", f, err)
            }
        }(file)
    }

    wg.Wait()
    close(errCh)

    // Collect errors
    var errs []error
    for err := range errCh {
        errs = append(errs, err)
    }

    if len(errs) > 0 {
        return fmt.Errorf("processing failed: %v", errs)
    }
    return nil
}

// Worker pool for file processing
func processFilesWithPool(files []string, workers int, fn func(string) error) error {
    jobs := make(chan string, len(files))
    results := make(chan error, len(files))

    // Start workers
    for i := 0; i < workers; i++ {
        go func() {
            for file := range jobs {
                results <- fn(file)
            }
        }()
    }

    // Send jobs
    for _, file := range files {
        jobs <- file
    }
    close(jobs)

    // Collect results
    for range files {
        if err := <-results; err != nil {
            return err
        }
    }

    return nil
}
```

## File Locking

```go
import "golang.org/x/sys/unix"

// Advisory file lock
func withFileLock(path string, fn func() error) error {
    file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
    if err != nil {
        return err
    }
    defer file.Close()

    // Acquire exclusive lock
    if err := unix.Flock(int(file.Fd()), unix.LOCK_EX); err != nil {
        return err
    }
    defer unix.Flock(int(file.Fd()), unix.LOCK_UN)

    return fn()
}

// Lock file for exclusive access
func lockFile(path string) (*os.File, error) {
    file, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0644)
    if err != nil {
        return nil, err
    }

    if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
        file.Close()
        return nil, fmt.Errorf("file is locked by another process")
    }

    return file, nil
}
```

## Summary

- **`os.ReadFile`/`os.WriteFile`** for simple read/write operations
- **`os.Open`/`os.Create`** with `defer Close()` for file handles
- **`io.Reader`/`io.Writer`** interfaces enable composable I/O
- **`bufio.Scanner`** for efficient line-by-line reading
- **`bufio.Writer`** with `Flush()` for buffered writing
- **`filepath`** package for cross-platform path manipulation
- **`encoding/json`** and **`encoding/csv`** for structured data
- **Always handle errors** and close resources with `defer`

---

## Exercises

1. **File Copy**: Implement a file copy function using `io.Copy`. Add progress reporting for large files.

2. **Line Counter**: Build a tool that counts lines, words, and characters in files (like `wc`).

3. **Log Parser**: Parse a log file and extract entries matching a pattern. Support multiple date formats.

4. **CSV Transformer**: Read a CSV, transform columns, and write to a new file.

5. **Directory Stats**: Walk a directory tree and report total size, file count, and largest files.

6. **Concurrent Grep**: Search multiple files for a pattern using worker goroutines.

7. **Atomic Write**: Implement atomic file write (write to temp, then rename).

8. **Config Watcher**: Watch a config file for changes and reload when modified.
