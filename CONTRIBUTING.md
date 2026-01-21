# Contributing to Podman Swarm

Thank you for considering contributing to Podman Swarm! This document provides guidelines for contributing to the project.

## 🚀 Getting Started

1. **Fork the Repository**
   ```bash
   git clone https://github.com/your-server-support/podman-swarm.git
   cd podman-swarm
   ```

2. **Set Up Development Environment**
   ```bash
   # Install Go 1.23+
   # Install Podman
   
   # Install dependencies
   go mod download
   
   # Build the project
   make build-all
   ```

3. **Create a Branch**
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/issue-description
   ```

## 📝 How to Contribute

### Reporting Bugs

Before creating bug reports, please check existing issues. When creating a bug report, include:

- **Clear title and description**
- **Steps to reproduce**
- **Expected behavior**
- **Actual behavior**
- **Environment details** (OS, Podman version, Go version)
- **Logs or error messages**

### Suggesting Features

Feature suggestions are welcome! Please:

- **Check TODO.md** to see if it's already planned
- **Describe the use case** clearly
- **Explain why this feature would be useful**
- **Provide examples** if possible

### Pull Requests

1. **Update Documentation**
   - Update README.md if needed
   - Add/update relevant .md files
   - Update TODO.md if implementing planned features

2. **Write Tests**
   - Add unit tests for new code
   - Ensure existing tests pass
   - Aim for >80% code coverage

3. **Follow Code Style**
   - Use `gofmt` to format code
   - Follow Go best practices
   - Add comments for exported functions
   - Use meaningful variable names

4. **Commit Messages**
   - Use clear, descriptive commit messages
   - Start with a verb (Add, Fix, Update, etc.)
   - Reference issues if applicable
   - Example: `Fix DNS resolution for services in non-default namespace (#42)`

5. **Submit PR**
   - Push to your fork
   - Create a pull request
   - Describe what your changes do
   - Link related issues

## 🧪 Testing

```bash
# Run unit tests
make test

# Run specific tests
go test ./internal/dns -v

# Check test coverage
go test -cover ./...
```

## 📚 Documentation

- Keep documentation up to date
- Use clear, concise language
- Provide examples where appropriate
- Update both English and Ukrainian versions

## 🔍 Code Review Process

1. Maintainers will review your PR
2. Address any feedback or requested changes
3. Once approved, your PR will be merged

## 🎯 Priority Areas

See [TODO.md](TODO.md) for current priorities. High-priority items include:

- Log streaming implementation
- Persistent storage support
- Unit and integration tests
- RBAC implementation
- Security enhancements

## 💡 Development Tips

- **Use CGO_ENABLED=0** when building to avoid CGO dependencies
- **Test in multi-node setup** using docker-compose.yml
- **Check linter errors** before committing
- **Use meaningful branch names** (feature/*, fix/*, docs/*)

## 📞 Communication

- **GitHub Issues**: For bugs and feature requests
- **Pull Requests**: For code contributions
- **Discussions**: For questions and general discussion

## ⚖️ License

By contributing, you agree that your contributions will be licensed under the same license as the project.

## 🙏 Thank You!

Every contribution helps make Podman Swarm better. We appreciate your time and effort!

---

For questions, open an issue or start a discussion on GitHub.
