# Contributing to DiffAI

Thank you for your interest in contributing to DiffAI! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Development Workflow](#development-workflow)
- [Code Standards](#code-standards)
- [Testing](#testing)
- [Pull Request Process](#pull-request-process)
- [Reporting Issues](#reporting-issues)
- [Security Vulnerabilities](#security-vulnerabilities)

## Code of Conduct

By participating in this project, you are expected to maintain a respectful and inclusive environment for all contributors.

## Getting Started

1. Fork the repository on GitHub
2. Clone your fork locally
3. Set up the development environment
4. Create a feature branch
5. Make your changes
6. Submit a pull request

## Development Setup

### Prerequisites

- **Go**: Version 1.25.4 or later (check `go.mod` for the current required version)
- **golangci-lint**: Version 2.6.2 (specified in `.tool-versions`)
- **Git**: For version control

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/klemjul/diffai.git
   cd diffai
   ```

2. Install dependencies:
   ```bash
   make install
   ```

3. Verify your setup by running tests:
   ```bash
   make test
   ```

## Development Workflow

### Making Changes

1. Create a feature branch from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes following the [Code Standards](#code-standards)

3. Format your code:
   ```bash
   make format
   ```

4. Run linters:
   ```bash
   make lint
   ```

5. Run tests:
   ```bash
   make test
   ```

6. Commit your changes with clear, descriptive commit messages:
   ```bash
   git commit -m "Add feature: description of your changes"
   ```

### Available Make Targets

- `make install` - Download Go module dependencies
- `make format` - Format Go code using `go fmt`
- `make lint` - Run golangci-lint
- `make test` - Run tests with coverage
- `make clean` - Tidy Go modules

## Code Standards

### Go Code Style

- Follow standard Go conventions and idioms
- Use `go fmt` for formatting (enforced by `make format`)
- Code must pass `golangci-lint` checks with the following linters:
  - `staticcheck` - Static analysis
  - `govet` - Go vet checks
  - `errcheck` - Check for unchecked errors

### Project Structure

```
diffai/
├── cmd/           # Command-line interface code
├── internal/      # Internal packages
│   ├── app/       # Application logic
│   ├── format/    # Formatting utilities
│   ├── git/       # Git integration
│   ├── llm/       # LLM client implementations
│   └── ui/        # User interface components
├── main.go        # Application entry point
└── makefile       # Build automation
```

### Best Practices

- Write clear, self-documenting code
- Add comments for complex logic
- Keep functions focused and small
- Handle errors appropriately
- Write tests for new functionality
- Ensure backward compatibility when possible

## Testing

### Running Tests

Run all tests:
```bash
make test
```

This generates a coverage report (`coverage.txt`).

### Writing Tests

- Place test files next to the code they test (e.g., `client.go` → `client_test.go`)
- Use table-driven tests where appropriate
- Aim for meaningful test coverage
- Test edge cases and error conditions
- Use the `testify` package for assertions (already in dependencies)

Example test structure:
```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"case1", "input1", "output1"},
        {"case2", "input2", "output2"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := YourFunction(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

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
   - Fill out the pull request template with:
     - Clear description of changes
     - Motivation for the changes
     - Any related issues
     - Testing performed

3. **After submission:**
   - CI checks must pass (tests, linting, CodeQL, Scorecard)
   - Address review feedback promptly
   - Keep your PR up to date with the main branch

4. **Merge:**
   - Once approved and all checks pass, a maintainer will merge your PR
   - Your contribution will be included in the next release

### PR Guidelines

- Keep pull requests focused on a single feature or fix
- Write clear commit messages
- Link to relevant issues
- Be responsive to feedback
- Ensure CI passes before requesting review

## Reporting Issues

### Bug Reports

When reporting bugs, please include:

- **Description**: Clear description of the bug
- **Steps to Reproduce**: Detailed steps to reproduce the issue
- **Expected Behavior**: What you expected to happen
- **Actual Behavior**: What actually happened
- **Environment**:
  - OS (Linux, macOS, Windows)
  - Go version
  - DiffAI version
- **Additional Context**: Any other relevant information

### Feature Requests

When suggesting features:

- Describe the feature and its benefits
- Explain the use case
- Consider backward compatibility
- Be open to discussion and alternatives

## Security Vulnerabilities

**Do not report security vulnerabilities as public issues.**

Please report security issues privately through [GitHub Security Advisories](https://github.com/klemjul/diffai/security/advisories/new).

See [SECURITY.md](./SECURITY.md) for more information.

## Questions?

If you have questions about contributing:

- Check the [README.md](./README.md) for general project information
- Review existing issues and pull requests
- Open a new issue for discussion

## License

By contributing to DiffAI, you agree that your contributions will be licensed under the [MIT License](./LICENSE).

---

Thank you for contributing to DiffAI! 🎉
