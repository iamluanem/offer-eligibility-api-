# Contributing Guidelines

## Git Workflow

### Branch Strategy

We use **feature branches** for all new features and improvements. The workflow is:

1. **Create a feature branch** from `main`:
   ```bash
   git checkout main
   git pull origin main
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** and commit them:
   ```bash
   git add .
   git commit -m "feat: your feature description"
   ```

3. **Push your branch**:
   ```bash
   git push origin feature/your-feature-name
   ```

4. **Create a Pull Request** on GitHub to merge into `main`

### Commit Message Convention

We follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `style:` - Code style changes (formatting, etc.)
- `refactor:` - Code refactoring
- `test:` - Adding or updating tests
- `chore:` - Maintenance tasks

### Branch Naming

- `feature/` - New features (e.g., `feature/redis-cache`)
- `fix/` - Bug fixes (e.g., `fix/memory-leak`)
- `refactor/` - Code refactoring (e.g., `refactor/service-layer`)
- `docs/` - Documentation updates (e.g., `docs/api-docs`)

### Example Workflow

```bash
# Start working on a new feature
git checkout main
git pull origin main
git checkout -b feature/rate-limiting

# Make changes and commit
git add .
git commit -m "feat: implement rate limiting middleware"

# Push and create PR
git push origin feature/rate-limiting
```

## Code Style

- Follow Go conventions and best practices
- Run `go fmt` before committing
- Run `go test ./...` to ensure all tests pass
- Keep functions small and focused
- Add comments for exported functions and types

## Testing

- Write tests for all new features
- Ensure test coverage doesn't decrease
- Run `go test -v ./...` before committing

