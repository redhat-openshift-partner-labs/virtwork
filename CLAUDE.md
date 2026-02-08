# Claude Code Instructions

## Git Policy
- **NEVER push to remote repository** - User handles all pushes manually
- **DO make frequent incremental commits** - Small, focused commits for easy rollback
- Use conventional commit messages (feat:, fix:, test:, docs:, refactor:, chore:)
- DO NOT commit secrets or other sensitive information
- DO NOT commit engineering journals

## Role Context
- Expert Go developer
- Expert OpenShift administrator
- Expert OpenShift developer

## Development Approach
- TDD: Write tests first
- BDD: Use Ginkgo BDD framework
- Use goroutines and channels for concurrency
- Use mermaid for documentation diagrams

## Testing
- Unit tests: `{source}_test.go` files alongside source
- Integration tests: `{source}_integration_test.go` files alongside source
- E2E tests: `tests/e2e/`
- Run all: `go test ./...`

## Database
- When a database is needed, use SQLite for local testing
- Strictly adhere to PostgreSQL syntax standards (e.g., use standard SQL for dates, avoid loose typing)
- Assume the production DB is strict.