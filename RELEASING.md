# Releasing Waterflow

This document describes the process for releasing a new version of Waterflow.

## Version Numbering

Waterflow follows [Semantic Versioning 2.0.0](https://semver.org/):

```
MAJOR.MINOR.PATCH[-PRERELEASE]

Examples:
  1.0.0      - First stable release
  1.1.0      - New features, backward compatible
  1.1.1      - Bug fixes only
  2.0.0      - Breaking changes
  1.2.0-rc.1 - Release candidate
  1.2.0-beta.1 - Beta release
```

### Version Increment Guidelines

| Change Type | Version Bump | Example |
|-------------|--------------|---------|
| Breaking API changes | MAJOR | 1.0.0 → 2.0.0 |
| New features (backward compatible) | MINOR | 1.0.0 → 1.1.0 |
| Bug fixes | PATCH | 1.0.0 → 1.0.1 |
| Pre-release | PRERELEASE | 1.1.0 → 1.1.0-rc.1 |

## Release Process

### 1. Prepare Release

#### Update Version References

No manual version updates needed - versions are injected at build time via `-ldflags`.

#### Update CHANGELOG

Edit `CHANGELOG.md`:

```markdown
## [Unreleased]

### Added
- New feature X

### Changed
- Modified behavior Y

### Fixed
- Bug fix Z

### Deprecated
- Feature A (will be removed in v2.0)

### Removed
- Removed feature B

### Security
- Security fix C
```

Move `[Unreleased]` contents to new version section:

```markdown
## [1.1.0] - 2026-02-15

### Added
- New feature X

## [Unreleased]
```

#### Run Tests

```bash
# Run all tests
make test

# Run integration tests
make integration-test

# Run acceptance tests
make acceptance-test

# Check code quality
make lint
```

### 2. Create Release

#### Tag and Push

```bash
# Ensure you're on main branch with latest changes
git checkout main
git pull origin main

# Create annotated tag
git tag -a v1.1.0 -m "Release v1.1.0"

# Push tag to trigger release workflow
git push origin v1.1.0
```

### 3. Monitor Release

1. Go to [GitHub Actions](https://github.com/Websoft9/waterflow/actions)
2. Monitor the "Release" workflow
3. Verify all jobs complete successfully

### 4. Verify Release

#### GitHub Release

- [ ] Release created at [Releases](https://github.com/Websoft9/waterflow/releases)
- [ ] Release notes are correct
- [ ] All 15 binaries are uploaded
- [ ] `checksums.txt` is present

#### Binaries

```bash
# Download and verify
VERSION=v1.1.0
curl -LO https://github.com/Websoft9/waterflow/releases/download/${VERSION}/checksums.txt
curl -LO https://github.com/Websoft9/waterflow/releases/download/${VERSION}/waterflow-cli-linux-amd64

# Verify checksum
sha256sum -c checksums.txt --ignore-missing

# Test binary
chmod +x waterflow-cli-linux-amd64
./waterflow-cli-linux-amd64 version
```

#### Docker Images

```bash
# Pull images
docker pull websoft9/waterflow-server:v1.1.0
docker pull websoft9/waterflow-agent:v1.1.0

# Verify version
docker run --rm websoft9/waterflow-server:v1.1.0 --version
docker run --rm websoft9/waterflow-agent:v1.1.0 --version
```

#### Go Module

```bash
# Test go install
go install github.com/Websoft9/waterflow/cmd/waterflow-cli@v1.1.0

# Verify
waterflow-cli version
```

### 5. Post-Release

#### Announce Release

- Update documentation if needed
- Announce on relevant channels

#### Prepare for Next Version

```bash
# Create develop branch for next version
git checkout develop
git merge main
git push origin develop
```

## Release Checklist Template

```markdown
## Release Checklist: v1.1.0

### Pre-Release
- [ ] All tests pass (unit, integration, acceptance)
- [ ] CHANGELOG.md updated
- [ ] No blocking issues in milestone
- [ ] Documentation updated

### Release
- [ ] Tag created: `git tag -a v1.1.0 -m "Release v1.1.0"`
- [ ] Tag pushed: `git push origin v1.1.0`
- [ ] CI workflow completed successfully

### Post-Release Verification
- [ ] GitHub Release created with notes
- [ ] All 15 binaries uploaded
- [ ] checksums.txt present
- [ ] Docker Hub images available
  - [ ] websoft9/waterflow-server:v1.1.0
  - [ ] websoft9/waterflow-agent:v1.1.0
- [ ] GHCR images available
- [ ] Go module installable
- [ ] Binary --version shows correct version
- [ ] Installation script works

### Announce
- [ ] Release notes published
- [ ] Documentation updated
```

## Hotfix Releases

For urgent bug fixes:

```bash
# Create hotfix branch from tag
git checkout -b hotfix/1.1.1 v1.1.0

# Make fixes
git commit -m "fix: critical bug"

# Update CHANGELOG
# Merge to main
git checkout main
git merge hotfix/1.1.1

# Tag and release
git tag -a v1.1.1 -m "Hotfix release v1.1.1"
git push origin main v1.1.1

# Clean up
git branch -d hotfix/1.1.1
```

## Pre-Release Versions

For testing before stable release:

```bash
# Release candidate
git tag -a v1.2.0-rc.1 -m "Release candidate 1 for v1.2.0"
git push origin v1.2.0-rc.1

# Beta
git tag -a v1.2.0-beta.1 -m "Beta 1 for v1.2.0"
git push origin v1.2.0-beta.1
```

Pre-releases:
- Are marked as "Pre-release" on GitHub
- Don't update the `latest` tag on Docker Hub
- Allow early testing by users

## Rollback

If a release has critical issues:

### 1. Mark as Known Issue

Add to Release notes:
```
⚠️ **Known Issue**: Description of problem. 
Please use v1.0.0 until this is fixed.
```

### 2. Delete Release (if necessary)

```bash
# Delete tag locally and remotely
git tag -d v1.1.0
git push origin :refs/tags/v1.1.0

# Delete GitHub Release via UI
```

### 3. Release Hotfix

Follow the hotfix process above.

## Automation

### GitHub Actions Workflows

| Workflow | Trigger | Actions |
|----------|---------|---------|
| `ci.yml` | Push/PR | Lint, test, build |
| `docker.yml` | Push, Tag | Build and push Docker images |
| `release.yml` | Tag v*.*.* | Build binaries, create release |

### Release Artifacts

The release workflow automatically:
1. Validates code (lint, test)
2. Builds 15 binaries (matrix strategy)
3. Generates checksums
4. Creates GitHub Release
5. Generates release notes

## Troubleshooting

### Release Workflow Failed

1. Check workflow logs in Actions tab
2. Fix issues and delete failed tag
3. Re-create and push tag

### Docker Push Failed

1. Verify `DOCKER_USERNAME` and `DOCKER_PASSWORD` secrets
2. Check Docker Hub repository exists
3. Re-run workflow

### Missing Binaries

1. Check build matrix in `release.yml`
2. Verify all platforms compiled successfully
3. Check artifact upload step

## Contact

For release questions:
- Open an issue with `release` label
- Contact maintainers
