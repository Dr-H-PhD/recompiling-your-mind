# Appendix D: Symfony-to-Go Service Mapping

Detailed mappings from Symfony components to Go patterns.

## HttpFoundation → net/http

### Request Object

| Symfony | Go |
|---------|-----|
| `$request->getMethod()` | `r.Method` |
| `$request->getPathInfo()` | `r.URL.Path` |
| `$request->getUri()` | `r.URL.String()` |
| `$request->getScheme()` | `r.URL.Scheme` |
| `$request->getHost()` | `r.Host` |
| `$request->query->get('key')` | `r.URL.Query().Get("key")` |
| `$request->query->all()` | `r.URL.Query()` |
| `$request->request->get('key')` | `r.FormValue("key")` |
| `$request->request->all()` | `r.PostForm` (after `r.ParseForm()`) |
| `$request->headers->get('X-Key')` | `r.Header.Get("X-Key")` |
| `$request->headers->all()` | `r.Header` |
| `$request->cookies->get('name')` | Loop `r.Cookies()` or `r.Cookie("name")` |
| `$request->getContent()` | `io.ReadAll(r.Body)` |
| `$request->toArray()` | `json.NewDecoder(r.Body).Decode(&v)` |
| `$request->getSession()` | Use session library (gorilla/sessions) |
| `$request->attributes->get()` | `r.Context().Value(key)` |
| `$request->getClientIp()` | Parse `r.Header.Get("X-Forwarded-For")` or `r.RemoteAddr` |

### Response Object

| Symfony | Go |
|---------|-----|
| `new Response($body)` | `w.Write([]byte(body))` |
| `new Response($body, 201)` | `w.WriteHeader(201); w.Write(...)` |
| `$response->headers->set()` | `w.Header().Set(key, val)` |
| `new JsonResponse($data)` | `json.NewEncoder(w).Encode(data)` |
| `new RedirectResponse($url)` | `http.Redirect(w, r, url, http.StatusFound)` |
| `new BinaryFileResponse($path)` | `http.ServeFile(w, r, path)` |
| `$response->setStatusCode()` | `w.WriteHeader(code)` |

### Example: Full Handler

```php
// Symfony
#[Route('/users/{id}', methods: ['GET'])]
public function show(int $id, Request $request): Response
{
    $format = $request->query->get('format', 'json');
    $user = $this->userRepository->find($id);

    if (!$user) {
        throw $this->createNotFoundException();
    }

    return $this->json($user);
}
```

```go
// Go
func (h *UserHandler) Show(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.Atoi(r.PathValue("id"))
    if err != nil {
        http.Error(w, "invalid id", http.StatusBadRequest)
        return
    }

    format := r.URL.Query().Get("format")
    if format == "" {
        format = "json"
    }

    user, err := h.repo.Find(r.Context(), id)
    if errors.Is(err, ErrNotFound) {
        http.Error(w, "not found", http.StatusNotFound)
        return
    }
    if err != nil {
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}
```

## Serializer → encoding/json

### Basic Serialisation

```php
// Symfony
$json = $serializer->serialize($user, 'json');
$user = $serializer->deserialize($json, User::class, 'json');
```

```go
// Go
data, err := json.Marshal(user)
err := json.Unmarshal(data, &user)
```

### Serialisation Groups

```php
// Symfony
#[Groups(['public'])]
private string $email;

$json = $serializer->serialize($user, 'json', ['groups' => ['public']]);
```

```go
// Go: Use separate structs
type UserPublic struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
}

type UserPrivate struct {
    UserPublic
    Email string `json:"email"`
}

func (u User) ToPublic() UserPublic {
    return UserPublic{ID: u.ID, Name: u.Name}
}
```

### Custom Normalisers

```php
// Symfony
class MoneyNormalizer implements NormalizerInterface
{
    public function normalize($object, $format = null, array $context = [])
    {
        return ['amount' => $object->getAmount() / 100, 'currency' => $object->getCurrency()];
    }
}
```

```go
// Go: Implement json.Marshaler
func (m Money) MarshalJSON() ([]byte, error) {
    return json.Marshal(map[string]interface{}{
        "amount":   float64(m.Amount) / 100,
        "currency": m.Currency,
    })
}
```

## Validator → go-playground/validator

### Constraints Mapping

| Symfony | go-playground/validator |
|---------|-------------------------|
| `#[NotBlank]` | `validate:"required"` |
| `#[NotNull]` | `validate:"required"` |
| `#[Email]` | `validate:"email"` |
| `#[Length(min: 2, max: 50)]` | `validate:"min=2,max=50"` |
| `#[Range(min: 1, max: 100)]` | `validate:"min=1,max=100"` |
| `#[Positive]` | `validate:"gt=0"` |
| `#[Regex(pattern: '/^\d+$/')]` | Custom validator |
| `#[Url]` | `validate:"url"` |
| `#[Uuid]` | `validate:"uuid"` |
| `#[Valid]` | `validate:"dive"` (for nested) |

### Example

```php
// Symfony
class CreateUserInput
{
    #[NotBlank]
    #[Length(min: 2, max: 100)]
    public string $name;

    #[NotBlank]
    #[Email]
    public string $email;

    #[NotBlank]
    #[Length(min: 8)]
    public string $password;
}
```

```go
// Go
type CreateUserInput struct {
    Name     string `json:"name" validate:"required,min=2,max=100"`
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
}

var validate = validator.New()

func (input CreateUserInput) Validate() error {
    return validate.Struct(input)
}
```

## Security → Middleware Patterns

### Authentication

```php
// Symfony Security
#[IsGranted('ROLE_USER')]
public function profile(): Response
{
    $user = $this->getUser();
}
```

```go
// Go: Middleware
func authRequired(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user := getUserFromContext(r.Context())
        if user == nil {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}

// Apply to routes
mux.Handle("/profile", authRequired(http.HandlerFunc(profileHandler)))
```

### Voters

```php
// Symfony Voter
class PostVoter extends Voter
{
    protected function voteOnAttribute($attribute, $subject, TokenInterface $token): bool
    {
        $user = $token->getUser();
        return $subject->getAuthor() === $user;
    }
}
```

```go
// Go: Check in handler or middleware
func canEditPost(user *User, post *Post) bool {
    return post.AuthorID == user.ID
}

func editPost(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())
    post := getPost(r)

    if !canEditPost(user, post) {
        http.Error(w, "forbidden", http.StatusForbidden)
        return
    }
    // Edit post
}
```

## Messenger → Channels and Workers

### Message Dispatching

```php
// Symfony Messenger
$this->messageBus->dispatch(new OrderCreatedEvent($order));
```

```go
// Go: Channel
type OrderCreatedEvent struct {
    OrderID int
}

var events = make(chan OrderCreatedEvent, 100)

// Dispatch
events <- OrderCreatedEvent{OrderID: order.ID}

// Worker
go func() {
    for event := range events {
        handleOrderCreated(event)
    }
}()
```

### Message Handlers

```php
// Symfony
#[AsMessageHandler]
class OrderCreatedHandler
{
    public function __invoke(OrderCreatedEvent $event): void
    {
        // Handle
    }
}
```

```go
// Go: Worker function
func orderCreatedWorker(ctx context.Context, events <-chan OrderCreatedEvent) {
    for {
        select {
        case <-ctx.Done():
            return
        case event := <-events:
            handleOrderCreated(event)
        }
    }
}
```

## Cache → go-cache or Redis

### Basic Caching

```php
// Symfony Cache
$value = $cache->get('key', function (ItemInterface $item) {
    $item->expiresAfter(3600);
    return computeExpensiveValue();
});
```

```go
// Go: go-cache
import "github.com/patrickmn/go-cache"

var c = cache.New(5*time.Minute, 10*time.Minute)

func getValue(key string) (interface{}, error) {
    if val, found := c.Get(key); found {
        return val, nil
    }

    val := computeExpensiveValue()
    c.Set(key, val, time.Hour)
    return val, nil
}
```

### Redis Cache

```go
// Go: Redis
import "github.com/go-redis/redis/v8"

var rdb = redis.NewClient(&redis.Options{Addr: "localhost:6379"})

func getValue(ctx context.Context, key string) (string, error) {
    val, err := rdb.Get(ctx, key).Result()
    if err == redis.Nil {
        val = computeExpensiveValue()
        rdb.Set(ctx, key, val, time.Hour)
        return val, nil
    }
    return val, err
}
```

## EventDispatcher → Callbacks or Channels

### Event Dispatching

```php
// Symfony
$this->eventDispatcher->dispatch(new UserCreatedEvent($user));
```

```go
// Go: Callback pattern
type EventDispatcher struct {
    listeners map[string][]func(any)
}

func (d *EventDispatcher) Dispatch(name string, event any) {
    for _, listener := range d.listeners[name] {
        listener(event)
    }
}

func (d *EventDispatcher) AddListener(name string, fn func(any)) {
    d.listeners[name] = append(d.listeners[name], fn)
}

// Or: Channel-based
type UserCreatedEvent struct {
    User User
}

var userCreatedChan = make(chan UserCreatedEvent, 100)

// Dispatch
userCreatedChan <- UserCreatedEvent{User: user}

// Listen
go func() {
    for event := range userCreatedChan {
        sendWelcomeEmail(event.User)
    }
}()
```

## Console → cobra or flag

### Command Definition

```php
// Symfony Console
class ImportUsersCommand extends Command
{
    protected function configure(): void
    {
        $this->setName('app:import-users')
             ->addArgument('file', InputArgument::REQUIRED)
             ->addOption('dry-run', null, InputOption::VALUE_NONE);
    }

    protected function execute(InputInterface $input, OutputInterface $output): int
    {
        $file = $input->getArgument('file');
        $dryRun = $input->getOption('dry-run');
        // Import users
        return Command::SUCCESS;
    }
}
```

```go
// Go: cobra
var importCmd = &cobra.Command{
    Use:   "import-users [file]",
    Short: "Import users from file",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        file := args[0]
        dryRun, _ := cmd.Flags().GetBool("dry-run")

        // Import users
        return nil
    },
}

func init() {
    importCmd.Flags().Bool("dry-run", false, "Dry run mode")
    rootCmd.AddCommand(importCmd)
}
```
