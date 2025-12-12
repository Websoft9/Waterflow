# Contributing to Waterflow

Thank you for your interest in contributing to Waterflow! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Project Structure](#project-structure)
- [Getting Started](#getting-started)
- [Branch Strategy](#branch-strategy)
- [Commit Message Guidelines](#commit-message-guidelines)
- [Pull Request Process](#pull-request-process)
- [Development Workflow](#development-workflow)
- [Coding Standards](#coding-standards)

## Project Structure

This repository contains two main components:

- **`/src`** (Coming soon) - Waterflow core engine implementation
- **`.bmad/`** - BMAD Method workflow system and configurations
- **`.github/`** - GitHub templates, workflows, and project management configurations

When contributing, please ensure your changes are in the appropriate directory.

## Code of Conduct

This project adheres to a Code of Conduct. By participating, you are expected to uphold this code. Please report unacceptable behavior to security@websoft9.com.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/Waterflow.git`
3. Add upstream remote: `git remote add upstream https://github.com/Websoft9/Waterflow.git`
4. Create a new branch for your work (see [Branch Strategy](#branch-strategy))

## Branch Strategy

We follow the **Git Flow** branching model:

### Main Branches

- **`main`** - Production-ready code. Only accepts merges from `release/*` and `hotfix/*` branches
- **`develop`** - Integration branch for features. Default branch for development

### Supporting Branches

#### Feature Branches
- **Naming**: `feat/<feature-name>` or `feature/<feature-name>`
- **Branch from**: `develop`
- **Merge back to**: `develop`
- **Example**: `feat/yaml-parser`, `feat/workflow-engine`

#### Bugfix Branches
- **Naming**: `fix/<bug-name>` or `bugfix/<bug-name>`
- **Branch from**: `develop`
- **Merge back to**: `develop`
- **Example**: `fix/config-parsing`, `fix/memory-leak`

#### Hotfix Branches
- **Naming**: `hotfix/<version>` or `hotfix/<issue>`
- **Branch from**: `main`
- **Merge back to**: `main` AND `develop`
- **Example**: `hotfix/1.2.1`, `hotfix/critical-security-fix`

#### Release Branches
- **Naming**: `release/<version>`
- **Branch from**: `develop`
- **Merge back to**: `main` AND `develop`
- **Example**: `release/1.3.0`

### Branch Naming Rules

- Use lowercase letters
- Use hyphens to separate words
- Be descriptive but concise
- Include issue number when applicable: `feat/123-add-authentication`

## Commit Message Guidelines

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification.

### Format

```
<type>(<scope>): <description>
```

### Types

- **feat** - New feature
- **fix** - Bug fix
- **docs** - Documentation changes
- **style** - Code style (formatting, etc.)
- **refactor** - Code refactoring
- **perf** - Performance improvements
- **test** - Test changes
- **build** - Build system changes
- **ci** - CI configuration changes
- **chore** - Other changes

### Examples

```
feat(parser): add YAML validation
fix(api): resolve null pointer error
docs: update README installation steps
```

## Pull Request Process

1. **Create a PR** from your feature branch to `develop` (or `main` for hotfixes)

2. **Fill out the PR template** with all required information

3. **Ensure CI passes**:
   - All tests pass
   - Code passes linting
   - No new warnings
   - Coverage doesn't decrease

4. **Request review** from at least one maintainer

5. **Address feedback** by pushing new commits to your branch

6. **Squash commits** if requested (maintainers may squash on merge)

7. **PR title** must follow commit message convention:
   ```
   feat(scope): add new feature
   fix(scope): resolve bug
   ```

### PR Labels

PRs will be automatically labeled based on:
- **Size**: `size/xs`, `size/s`, `size/m`, `size/l`, `size/xl`
- **Type**: `type/feature`, `type/bug`, `type/documentation`, etc.
- **Area**: `area/ci`, `area/tests`, `area/documentation`, etc.

## Development Workflow

### 1. Start New Work

```bash
# Update your local repository
git checkout develop
git pull upstream develop

# Create feature branch
git checkout -b feat/my-new-feature

# Make changes
# ...

# Commit following commit guidelines
git commit -m "feat(scope): add new feature"
```

### 2. Keep Branch Updated

```bash
# Regularly sync with upstream
git fetch upstream
git rebase upstream/develop
```

### 3. Push and Create PR

```bash
# Push to your fork
git push origin feat/my-new-feature

# Create PR through GitHub UI
```

### 4. After PR is Merged

```bash
# Update local develop
git checkout develop
git pull upstream develop

# Delete feature branch
git branch -d feat/my-new-feature
git push origin --delete feat/my-new-feature
```

## Coding Standards

### Go Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Use `golangci-lint` for linting
- Write meaningful comments for exported functions
- Keep functions small and focused
- Write table-driven tests

### Testing

- Write unit tests for new features
- Maintain or improve code coverage
- Use meaningful test names: `TestFunctionName_Scenario_ExpectedBehavior`
- Mock external dependencies

### Documentation

- Update README.md if adding user-facing features
- Add inline documentation for complex logic
- Update API documentation if changing interfaces
- Include examples where helpful

## Questions?

Feel free to:
- Open a [Discussion](https://github.com/Websoft9/Waterflow/discussions)
- Ask in an [Issue](https://github.com/Websoft9/Waterflow/issues)
- Contact the maintainers

Thank you for contributing to Waterflow! 🚀
