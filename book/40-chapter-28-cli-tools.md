# Chapter 28: Building CLI Tools

PHP developers build CLI tools using Symfony Console, with commands, arguments, options, and interactive prompts. Go's standard library provides simpler primitives that compile to fast, single-binary executables.

## Why Go for CLI Tools

Go excels at CLI tools for several reasons:

- **Single binary**: No runtime dependencies, no PHP interpreter needed
- **Cross-compilation**: Build for Linux, macOS, and Windows from any machine
- **Fast startup**: Milliseconds, not the 50-100ms PHP requires for autoloading
- **Small binaries**: Typically 5-15MB, easy to distribute

Compare deploying a Symfony Console application versus a Go binary:

```
# Symfony Console
composer install
php bin/console my:command

# Go
./mycli  # That's it
```

## The `flag` Package

PHP CLI argument parsing:

```php
// Manual parsing
$options = getopt('n:a:', ['name:', 'age:']);
$name = $options['n'] ?? $options['name'] ?? 'Guest';

// Or Symfony Console
protected function configure(): void
{
    $this->addOption('name', 'n', InputOption::VALUE_REQUIRED, 'User name', 'Guest');
    $this->addArgument('file', InputArgument::REQUIRED, 'Input file');
}
```

Go's `flag` package:

```go
package main

import (
    "flag"
    "fmt"
)

func main() {
    // Define flags
    name := flag.String("name", "Guest", "Name of the user")
    age := flag.Int("age", 0, "Age of the user")
    verbose := flag.Bool("verbose", false, "Enable verbose output")

    // Parse command line
    flag.Parse()

    // Use the values (flags are pointers)
    fmt.Printf("Hello, %s! Age: %d\n", *name, *age)
    if *verbose {
        fmt.Println("Verbose mode enabled")
    }

    // Positional arguments (after flags)
    args := flag.Args()
    fmt.Printf("Additional arguments: %v\n", args)
}
```

Usage:

```bash
./mycli -name Alice -age 30 -verbose file1.txt file2.txt
# Hello, Alice! Age: 30
# Verbose mode enabled
# Additional arguments: [file1.txt file2.txt]
```

### Using Variables Instead of Pointers

```go
var (
    name    string
    age     int
    verbose bool
)

func init() {
    flag.StringVar(&name, "name", "Guest", "Name of the user")
    flag.IntVar(&age, "age", 0, "Age of the user")
    flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
}

func main() {
    flag.Parse()
    fmt.Printf("Hello, %s!\n", name)  // No pointer dereference
}
```

### Short and Long Options

The standard `flag` package doesn't support short options (`-n` vs `--name`). For that, use `pflag`:

```go
import flag "github.com/spf13/pflag"

func main() {
    var name string
    flag.StringVarP(&name, "name", "n", "Guest", "Name of the user")
    flag.Parse()
}
```

Usage: `./mycli -n Alice` or `./mycli --name Alice`

### Custom Usage Message

```go
func main() {
    flag.Usage = func() {
        fmt.Fprintf(os.Stderr, "Usage: %s [options] <files...>\n\n", os.Args[0])
        fmt.Fprintln(os.Stderr, "A tool for processing files.\n")
        fmt.Fprintln(os.Stderr, "Options:")
        flag.PrintDefaults()
        fmt.Fprintln(os.Stderr, "\nExamples:")
        fmt.Fprintln(os.Stderr, "  mycli -verbose file.txt")
        fmt.Fprintln(os.Stderr, "  mycli --name Alice *.json")
    }

    flag.Parse()

    if flag.NArg() == 0 {
        flag.Usage()
        os.Exit(1)
    }
}
```

## Subcommands

Symfony Console organises functionality into commands:

```php
class App extends Application
{
    protected function configure(): void
    {
        $this->add(new UserCreateCommand());
        $this->add(new UserDeleteCommand());
    }
}
// Usage: php bin/console user:create
```

Go handles subcommands explicitly:

```go
package main

import (
    "flag"
    "fmt"
    "os"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: mycli <command> [options]")
        fmt.Println("Commands: create, delete, list")
        os.Exit(1)
    }

    switch os.Args[1] {
    case "create":
        createCmd(os.Args[2:])
    case "delete":
        deleteCmd(os.Args[2:])
    case "list":
        listCmd(os.Args[2:])
    default:
        fmt.Printf("Unknown command: %s\n", os.Args[1])
        os.Exit(1)
    }
}

func createCmd(args []string) {
    fs := flag.NewFlagSet("create", flag.ExitOnError)
    name := fs.String("name", "", "Name to create")
    fs.Parse(args)

    if *name == "" {
        fmt.Println("Error: -name is required")
        fs.Usage()
        os.Exit(1)
    }

    fmt.Printf("Creating: %s\n", *name)
}

func deleteCmd(args []string) {
    fs := flag.NewFlagSet("delete", flag.ExitOnError)
    id := fs.Int("id", 0, "ID to delete")
    force := fs.Bool("force", false, "Skip confirmation")
    fs.Parse(args)

    if *id == 0 {
        fmt.Println("Error: -id is required")
        os.Exit(1)
    }

    if !*force {
        fmt.Printf("Delete ID %d? [y/N]: ", *id)
        var confirm string
        fmt.Scanln(&confirm)
        if confirm != "y" && confirm != "Y" {
            fmt.Println("Cancelled")
            return
        }
    }

    fmt.Printf("Deleted: %d\n", *id)
}

func listCmd(args []string) {
    fs := flag.NewFlagSet("list", flag.ExitOnError)
    limit := fs.Int("limit", 10, "Number of items")
    fs.Parse(args)

    fmt.Printf("Listing %d items...\n", *limit)
}
```

Usage:

```bash
./mycli create -name "Alice"
./mycli delete -id 42 -force
./mycli list -limit 5
```

### Using Cobra for Complex CLIs

For sophisticated CLIs, use Cobra (powers kubectl, hugo, gh):

```go
import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
    Use:   "mycli",
    Short: "A tool for managing things",
}

var createCmd = &cobra.Command{
    Use:   "create [name]",
    Short: "Create a new item",
    Args:  cobra.ExactArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        name := args[0]
        verbose, _ := cmd.Flags().GetBool("verbose")
        fmt.Printf("Creating: %s (verbose: %v)\n", name, verbose)
    },
}

func init() {
    createCmd.Flags().BoolP("verbose", "v", false, "Verbose output")
    rootCmd.AddCommand(createCmd)
}

func main() {
    rootCmd.Execute()
}
```

## User Input and Output

### Reading Input

```go
import (
    "bufio"
    "fmt"
    "os"
    "strings"
)

// Simple prompt
func prompt(message string) string {
    fmt.Print(message)
    reader := bufio.NewReader(os.Stdin)
    input, _ := reader.ReadString('\n')
    return strings.TrimSpace(input)
}

// Yes/No confirmation
func confirm(message string) bool {
    response := prompt(message + " [y/N]: ")
    return strings.ToLower(response) == "y"
}

// Password input (no echo)
import "golang.org/x/term"

func promptPassword(message string) string {
    fmt.Print(message)
    password, _ := term.ReadPassword(int(os.Stdin.Fd()))
    fmt.Println()
    return string(password)
}

// Usage
func main() {
    name := prompt("Enter name: ")
    password := promptPassword("Enter password: ")

    if confirm("Save these credentials?") {
        fmt.Println("Saved!")
    }
}
```

### Coloured Output

```go
import "github.com/fatih/color"

func main() {
    // Simple colours
    color.Red("Error: something went wrong")
    color.Green("Success!")
    color.Yellow("Warning: check your input")

    // Styled output
    bold := color.New(color.Bold)
    bold.Println("Important message")

    success := color.New(color.FgGreen, color.Bold)
    success.Println("Operation completed")

    // Sprintf variants
    msg := color.RedString("Error: %s", err)
    fmt.Println(msg)
}
```

### Progress Indicators

```go
import "github.com/schollz/progressbar/v3"

func processFiles(files []string) {
    bar := progressbar.Default(int64(len(files)))

    for _, file := range files {
        processFile(file)
        bar.Add(1)
    }
}

// Spinner for indeterminate progress
import "github.com/briandowns/spinner"

func longOperation() {
    s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
    s.Suffix = " Processing..."
    s.Start()

    // Do work
    time.Sleep(3 * time.Second)

    s.Stop()
    fmt.Println("Done!")
}
```

### Tabular Output

```go
import "github.com/olekukonko/tablewriter"

func printUsers(users []User) {
    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"ID", "Name", "Email", "Active"})

    for _, u := range users {
        active := "No"
        if u.Active {
            active = "Yes"
        }
        table.Append([]string{
            fmt.Sprintf("%d", u.ID),
            u.Name,
            u.Email,
            active,
        })
    }

    table.Render()
}

// Output:
// +----+-------+------------------+--------+
// | ID | NAME  | EMAIL            | ACTIVE |
// +----+-------+------------------+--------+
// |  1 | Alice | alice@example.com | Yes   |
// |  2 | Bob   | bob@example.com   | No    |
// +----+-------+------------------+--------+
```

## Error Handling and Exit Codes

```go
package main

import (
    "fmt"
    "os"
)

const (
    ExitSuccess = 0
    ExitError   = 1
    ExitUsage   = 2
)

func main() {
    if err := run(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(ExitError)
    }
}

func run() error {
    // Parse flags
    flag.Parse()

    if flag.NArg() == 0 {
        flag.Usage()
        os.Exit(ExitUsage)
    }

    // Do work
    for _, file := range flag.Args() {
        if err := processFile(file); err != nil {
            return fmt.Errorf("processing %s: %w", file, err)
        }
    }

    return nil
}
```

### Graceful Shutdown

```go
import (
    "context"
    "os"
    "os/signal"
    "syscall"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())

    // Handle interrupt signals
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

    go func() {
        <-sigCh
        fmt.Println("\nShutting down...")
        cancel()
    }()

    if err := run(ctx); err != nil {
        if err == context.Canceled {
            fmt.Println("Operation cancelled")
            os.Exit(0)
        }
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}

func run(ctx context.Context) error {
    for i := 0; i < 100; i++ {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            // Do work
            processItem(i)
        }
    }
    return nil
}
```

## Configuration Files

### Reading JSON Config

```go
type Config struct {
    Server   string `json:"server"`
    Port     int    `json:"port"`
    Verbose  bool   `json:"verbose"`
    Timeout  int    `json:"timeout"`
}

func loadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var config Config
    if err := json.Unmarshal(data, &config); err != nil {
        return nil, err
    }

    return &config, nil
}

// With defaults
func loadConfigWithDefaults(path string) *Config {
    config := &Config{
        Server:  "localhost",
        Port:    8080,
        Timeout: 30,
    }

    data, err := os.ReadFile(path)
    if err != nil {
        return config  // Return defaults
    }

    json.Unmarshal(data, config)  // Override with file values
    return config
}
```

### Using Viper for Complex Config

```go
import "github.com/spf13/viper"

func initConfig() {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath(".")
    viper.AddConfigPath("$HOME/.mycli")

    // Environment variables override config file
    viper.SetEnvPrefix("MYCLI")
    viper.AutomaticEnv()

    // Defaults
    viper.SetDefault("server", "localhost")
    viper.SetDefault("port", 8080)

    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            log.Fatal(err)
        }
    }
}

func main() {
    initConfig()

    server := viper.GetString("server")
    port := viper.GetInt("port")
    verbose := viper.GetBool("verbose")

    fmt.Printf("Connecting to %s:%d\n", server, port)
}
```

## Testing CLI Applications

### Unit Testing Functions

```go
func TestProcessFile(t *testing.T) {
    // Create temp file
    content := "test content"
    tmpfile, _ := os.CreateTemp("", "test*.txt")
    tmpfile.WriteString(content)
    tmpfile.Close()
    defer os.Remove(tmpfile.Name())

    // Test
    result, err := processFile(tmpfile.Name())
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result != expected {
        t.Errorf("got %v, want %v", result, expected)
    }
}
```

### Integration Testing with exec.Command

```go
func TestCLI(t *testing.T) {
    // Build the binary
    cmd := exec.Command("go", "build", "-o", "testcli", ".")
    if err := cmd.Run(); err != nil {
        t.Fatalf("build failed: %v", err)
    }
    defer os.Remove("testcli")

    tests := []struct {
        args     []string
        wantOut  string
        wantErr  bool
        wantCode int
    }{
        {
            args:    []string{"-name", "Alice"},
            wantOut: "Hello, Alice!",
        },
        {
            args:     []string{"-invalid"},
            wantErr:  true,
            wantCode: 2,
        },
    }

    for _, tt := range tests {
        cmd := exec.Command("./testcli", tt.args...)
        output, err := cmd.CombinedOutput()

        if tt.wantErr && err == nil {
            t.Errorf("expected error for args %v", tt.args)
        }

        if !strings.Contains(string(output), tt.wantOut) {
            t.Errorf("output %q doesn't contain %q", output, tt.wantOut)
        }
    }
}
```

### Testing with Captured Output

```go
func TestOutput(t *testing.T) {
    // Capture stdout
    old := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    // Run function
    printGreeting("Alice")

    // Restore and read
    w.Close()
    os.Stdout = old

    var buf bytes.Buffer
    io.Copy(&buf, r)

    if !strings.Contains(buf.String(), "Hello, Alice") {
        t.Errorf("unexpected output: %s", buf.String())
    }
}
```

## Complete Example: File Processing Tool

```go
package main

import (
    "bufio"
    "flag"
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

var (
    version = "1.0.0"
    pattern string
    output  string
    verbose bool
)

func init() {
    flag.StringVar(&pattern, "pattern", "", "Search pattern (required)")
    flag.StringVar(&output, "output", "", "Output file (default: stdout)")
    flag.BoolVar(&verbose, "verbose", false, "Verbose output")

    flag.Usage = func() {
        fmt.Fprintf(os.Stderr, "grep-lite v%s - Simple pattern search\n\n", version)
        fmt.Fprintf(os.Stderr, "Usage: %s [options] <files...>\n\n", os.Args[0])
        fmt.Fprintln(os.Stderr, "Options:")
        flag.PrintDefaults()
    }
}

func main() {
    flag.Parse()

    if pattern == "" {
        fmt.Fprintln(os.Stderr, "Error: -pattern is required")
        flag.Usage()
        os.Exit(2)
    }

    if flag.NArg() == 0 {
        fmt.Fprintln(os.Stderr, "Error: at least one file is required")
        flag.Usage()
        os.Exit(2)
    }

    // Expand globs
    var files []string
    for _, arg := range flag.Args() {
        matches, err := filepath.Glob(arg)
        if err != nil || len(matches) == 0 {
            files = append(files, arg)
        } else {
            files = append(files, matches...)
        }
    }

    // Set up output
    var out *os.File
    if output == "" {
        out = os.Stdout
    } else {
        var err error
        out, err = os.Create(output)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error creating output: %v\n", err)
            os.Exit(1)
        }
        defer out.Close()
    }

    // Process files
    totalMatches := 0
    for _, file := range files {
        matches, err := searchFile(file, pattern, out)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", file, err)
            continue
        }
        totalMatches += matches
    }

    if verbose {
        fmt.Fprintf(os.Stderr, "Found %d matches in %d files\n", totalMatches, len(files))
    }
}

func searchFile(path, pattern string, out *os.File) (int, error) {
    file, err := os.Open(path)
    if err != nil {
        return 0, err
    }
    defer file.Close()

    matches := 0
    scanner := bufio.NewScanner(file)
    lineNum := 0

    for scanner.Scan() {
        lineNum++
        line := scanner.Text()
        if strings.Contains(line, pattern) {
            fmt.Fprintf(out, "%s:%d: %s\n", path, lineNum, line)
            matches++
        }
    }

    if verbose && matches > 0 {
        fmt.Fprintf(os.Stderr, "%s: %d matches\n", path, matches)
    }

    return matches, scanner.Err()
}
```

## Summary

- **`flag` package** handles command-line parsing simply
- **FlagSets** enable subcommand patterns
- **Cobra** provides advanced CLI features (kubectl, hugo style)
- **User input** uses `bufio.Reader` and `term` for passwords
- **Coloured output** via `fatih/color`
- **Progress bars** with `progressbar` library
- **Exit codes** follow Unix conventions (0 success, 1 error, 2 usage)
- **Testing** combines unit tests with integration via `exec.Command`

---

## Exercises

1. **Basic CLI**: Build a CLI that accepts `-input` and `-output` flags, reads a file, transforms it, and writes the result.

2. **Subcommands**: Create a task manager CLI with `add`, `list`, `done`, and `delete` subcommands.

3. **Interactive Mode**: Build a REPL (Read-Eval-Print Loop) that accepts commands interactively.

4. **Progress Bar**: Create a file copier with progress indication for large files.

5. **Config Integration**: Build a CLI that reads settings from a config file, environment variables, and flags (in that priority order).

6. **Coloured Output**: Create a log viewer that colours ERROR lines red, WARN yellow, and INFO green.

7. **Testing Suite**: Write comprehensive tests for a CLI including unit tests and integration tests.

8. **Cross-Platform**: Build a CLI and cross-compile for Linux, macOS, and Windows. Test on each platform.
