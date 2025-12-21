# Contributing to DiffAI

## Getting Started

### Prerequisites

- **Go**: Check [go.mod](go.mod) for the required version
- **golangci-lint**: Check [.tool-versions](.tool-versions) for the required version

### Installation

1. Fork and clone the repository:

2. Install dependencies:
   ```bash
   make install
   ```

### Testing

Run the test suite:
```bash
make test
```

### Linting

Check code quality:
```bash
make lint
```

### Format

Format your code:
```bash
make format
```

## Guidelines

- Write clear, self-documenting code
- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Handle errors explicitly
- Write tests for new functionality
- Use meaningful commit messages that follow [conventional commit](https://www.conventionalcommits.org/en/v1.0.0/)

## Pull Request Process

1. **Before submitting:**
   - Ensure all tests pass (`make test`)
   - Ensure code is formatted (`make format`)
   - Ensure linting passes (`make lint`)
   - Ensure `go.mod` is tidy (`make clean` and verify no changes)
   - Update documentation if needed

2. **Submit your PR:**
   - Push your branch to your fork
   - Create a pull request against the `main` branch
   - Fill out follow the [pull request template](.github/PULL_REQUEST_TEMPLATE/pull_request_template.md)

3. **After submission:**
   - CI checks must pass
   - Try to Address review feedback promptly
   - Keep your PR up to date with the main branch

4. **Merge:**
   - Once approved and all checks pass, a maintainer will merge your PR

## License

By contributing to DiffAI, you agree that your contributions will be licensed under the [MIT License](./LICENSE).

---

Thank you for your interest in contributing to DiffAI!