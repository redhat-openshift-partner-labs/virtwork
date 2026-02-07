# Claude Code Instructions

## Git Policy
- **NEVER push to remote repository** - User handles all pushes manually
- **DO make frequent incremental commits** - Small, focused commits for easy rollback
- Use conventional commit messages (feat:, fix:, test:, docs:, refactor:, chore:)

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
- Unit tests: `*_test.go` files alongside source
- Integration tests: `tests/integration/`
- E2E tests: `tests/e2e/`
- Run all: `go test ./...`