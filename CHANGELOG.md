## v0.4.0 (2025-09-06)

### Feat

- add more debug logging to help narrow down issues

### Fix

- **deps**: update module github.com/spf13/cobra to v1.10.1

## v0.3.0 (2025-09-03)

### Feat

- add support for seconds in patterns

### Fix

- remove all references to logger test mode, no longer used
- clean up log sync error handling

### Perf

- replace fmt.*printf uses with more performant options

## v0.2.0 (2025-09-03)

### Feat

- **deps**: bump go minimum version to 1.24.5

### Fix

- **deps**: update module github.com/stretchr/testify to v1.11.1
- **deps**: update module golang.org/x/sys to v0.35.0
- **deps**: update module golang.org/x/sys to v0.34.0
- **docs**: Add missing docstring for go_release_package

### Refactor

- sort golangci-lint settings
- sort revive rules
- split util package based on usage to reduce build complexity

## v0.1.3 (2025-05-28)

### Feat

- add multi-platform builds and checksum generation for apply-retention-policy in release workflow

### Refactor

- combine push runs into one command

## v0.1.2 (2025-05-28)

### Fix

- add name argument to satisfy buildifier

### Refactor

- disable image builds on non-linux systems due to errors
- update Bazel build configuration to streamline image creation and enhance platform support

## v0.1.1 (2025-05-26)

### Feat

- **dev**: add Bazel configuration rules for consistent usage
- enhance file manager with regex pattern parsing and add unit tests

### Fix

- address gosec issues in files_darwin
- whitespace error in files package docs
- manage golangci-lint via bazel
- define all zstd toolchains
- remove unused bazel import in build file
- apply linter fixes

### Refactor

- simplify parameters in setupTestFile function for improved readability
- modernize Go code with context support and improved patterns

## v0.1.0 (2025-05-12)
