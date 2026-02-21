# PHP to Go Migration Example

This project demonstrates the Strangler Fig pattern for migrating from PHP to Go.

## Structure

```
06-migration-example/
├── php/              # Legacy PHP application
│   ├── public/
│   ├── src/
│   └── composer.json
├── go/               # New Go service
│   ├── main.go
│   └── go.mod
├── nginx/            # Load balancer config
│   └── nginx.conf
└── docker-compose.yml
```

## Migration Strategy

1. **Phase 1**: All traffic to PHP
2. **Phase 2**: Route new endpoints to Go
3. **Phase 3**: Migrate existing endpoints gradually
4. **Phase 4**: Retire PHP

## Running

```bash
docker-compose up
```

## Endpoints

| Endpoint      | PHP | Go  | Notes                    |
|---------------|-----|-----|--------------------------|
| /api/v1/*     | ✓   |     | Legacy API               |
| /api/v2/*     |     | ✓   | New API in Go            |
| /health       |     | ✓   | Health check             |
| /legacy/*     | ✓   |     | To be migrated           |

## Shared Resources

- Database: Both PHP and Go connect to the same PostgreSQL
- Auth: Shared JWT tokens (same secret)
- Session: Redis-backed, accessible from both
