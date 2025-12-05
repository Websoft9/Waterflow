# Contributing to Waterflow

Thank you for your interest in contributing to Waterflow! We welcome contributions from the community and are grateful for your help in making this project better.

## 🤝 How to Contribute

### Ways to Contribute

- **🐛 Report Bugs**: Found a bug? [Open an issue](https://github.com/Websoft9/Waterflow/issues/new?template=bug_report.md)
- **💡 Suggest Features**: Have an idea? [Open a feature request](https://github.com/Websoft9/Waterflow/issues/new?template=feature_request.md)
- **📝 Improve Documentation**: Help make our docs clearer and more comprehensive
- **🔧 Submit Code**: Fix bugs, add features, or improve performance
- **🧪 Write Tests**: Help ensure code quality and prevent regressions
- **📢 Spread the Word**: Share Waterflow with others who might find it useful

### Development Workflow

We follow the [BMAD Method](https://github.com/bmad-code-org/BMAD-METHOD) for AI-driven agile development. Here's our contribution process:

#### 1. Choose an Issue

- Check our [GitHub Issues](https://github.com/Websoft9/Waterflow/issues) for open tasks
- Look for issues labeled `good first issue` or `help wanted`
- Comment on the issue to indicate you're working on it

#### 2. Fork and Clone

```bash
# Fork the repository on GitHub
# Clone your fork
git clone https://github.com/YOUR_USERNAME/Waterflow.git
cd Waterflow

# Add upstream remote
git remote add upstream https://github.com/Websoft9/Waterflow.git
```

#### 3. Create a Feature Branch

```bash
# Create and switch to a new branch
git checkout -b feature/your-feature-name
# or for bug fixes
git checkout -b fix/issue-number-description
```

#### 4. Set Up Development Environment

```bash
# Install dependencies (when available)
make dev-setup
# or
npm install  # if using Node.js
# or
pip install -r requirements.txt  # if using Python
# or
go mod download  # if using Go
```

#### 5. Make Your Changes

- Write clear, focused commits
- Follow our coding standards (see below)
- Add tests for new functionality
- Update documentation as needed

#### 6. Test Your Changes

```bash
# Run tests
make test
# or
npm test
# or
python -m pytest
# or
go test ./...

# Run linting
make lint
# or
npm run lint

# Build the project
make build
```

#### 7. Submit a Pull Request

```bash
# Ensure you're up to date with upstream
git fetch upstream
git rebase upstream/main

# Push your branch
git push origin feature/your-feature-name
```

Then create a Pull Request on GitHub with:
- Clear title describing the change
- Detailed description of what was changed and why
- Reference to any related issues
- Screenshots/videos for UI changes

## 📋 Pull Request Guidelines

### Before Submitting

- [ ] Tests pass locally
- [ ] Code follows our style guidelines
- [ ] Documentation is updated
- [ ] Commit messages are clear and descriptive
- [ ] Branch is up to date with main

### PR Template

Please use our [Pull Request Template](.github/PULL_REQUEST_TEMPLATE.md) when creating PRs.

### Review Process

1. **Automated Checks**: CI/CD pipeline runs tests and linting
2. **Code Review**: At least one maintainer reviews your code
3. **Approval**: PR is approved and merged
4. **Release**: Changes are included in the next release

## 🛠️ Development Setup

### Prerequisites

- Git
- Docker (for containerized development)
- Make (build tool)
- Go 1.21+ or Python 3.9+ or Node.js 18+ (depending on implementation choice)

### Local Development

```bash
# Clone the repository
git clone https://github.com/Websoft9/Waterflow.git
cd Waterflow

# Set up development environment
make dev-setup

# Start development server (when available)
make dev

# Run tests in watch mode
make test-watch
```

### Using Dev Containers

We provide a Dev Container configuration for consistent development environments:

1. Open in VS Code
2. When prompted, click "Reopen in Container"
3. All dependencies will be automatically installed

## 📝 Coding Standards

### General Guidelines

- **Language**: Follow the language-specific conventions below
- **Documentation**: Document public APIs and complex logic
- **Testing**: Write tests for all new functionality
- **Security**: Follow secure coding practices
- **Performance**: Consider performance implications of changes

### Go (if chosen as primary language)

```go
// Follow standard Go conventions
// Use gofmt for formatting
// Follow effective Go guidelines
// Use meaningful variable and function names
```

### Python (if chosen as primary language)

```python
# Follow PEP 8
# Use type hints for public APIs
# Write docstrings for modules, classes, and functions
# Use meaningful variable names
```

### YAML Configuration

```yaml
# Use consistent indentation (2 spaces)
# Add comments for complex configurations
# Use anchors and aliases to reduce duplication
# Validate against JSON Schema
```

## 🧪 Testing

### Test Categories

- **Unit Tests**: Test individual functions and methods
- **Integration Tests**: Test component interactions
- **End-to-End Tests**: Test complete workflows
- **Performance Tests**: Ensure performance requirements are met

### Running Tests

```bash
# Run all tests
make test

# Run specific test categories
make test-unit
make test-integration
make test-e2e

# Run tests with coverage
make test-coverage
```

### Writing Tests

- Use descriptive test names
- Test both positive and negative cases
- Mock external dependencies
- Include edge cases and error conditions

## 📚 Documentation

### Types of Documentation

- **README.md**: Project overview and quick start
- **API Documentation**: Generated from code comments
- **User Guides**: Step-by-step tutorials
- **Architecture Docs**: System design and patterns

### Documentation Standards

- Use Markdown for all documentation
- Include code examples where helpful
- Keep screenshots up to date
- Use consistent terminology

## 🔒 Security

- Never commit secrets or credentials
- Use environment variables for configuration
- Follow secure coding practices
- Report security issues via [SECURITY.md](SECURITY.md)

## 🤖 AI-Assisted Development

This project embraces AI-driven development using the BMAD Method. When using AI coding assistants:

- Review and understand generated code
- Ensure code follows project standards
- Add appropriate tests
- Document any complex logic
- Verify functionality manually

## 📞 Getting Help

- **Documentation**: Check our [docs/](docs/) directory
- **Issues**: Search existing [GitHub Issues](https://github.com/Websoft9/Waterflow/issues)
- **Discussions**: Join [GitHub Discussions](https://github.com/Websoft9/Waterflow/discussions)
- **Community**: Connect with other contributors

## 📄 License

By contributing to Waterflow, you agree that your contributions will be licensed under the same [MIT License](LICENSE) that covers the project.

## 🙏 Recognition

Contributors are recognized in our [CHANGELOG.md](CHANGELOG.md) and release notes. Significant contributions may also be acknowledged in the main README.

Thank you for contributing to Waterflow! 🚀