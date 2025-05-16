# Apply Retention Policy

A command-line tool to apply retention policies to backup files. It helps manage disk space by automatically deleting old backup files while maintaining a specified number of hourly, daily, weekly, monthly, and yearly backups.

This project is both a useful tool that I needed and an experiment in AI-assisted development using Cursor.

## Features

- Configurable retention periods (hourly, daily, weekly, monthly, yearly)
- Flexible file pattern matching
- Dry run mode for safe testing
- Structured logging
- Docker support

## Installation

### Binary

Download the latest release from the [releases page](https://github.com/TotallyNotRobots/apply-retention-policy/releases).

### Docker

```bash
docker pull ghcr.io/totallynotrobots/apply-retention-policy:latest
```

## Usage

1. Create a configuration file (see `configs/example.yaml` for an example):

```yaml
retention:
  hourly: 24
  daily: 7
  weekly: 4
  monthly: 12
  yearly: 5

file_pattern: "backup-{year}-{month}-{day}-{hour}-{minute}.tar.gz"
directory: "/path/to/backups"
log_level: "info"
dry_run: false
```

2. Run the tool:

```bash
# Using binary
./apply-retention-policy prune --config config.yaml

# Using Docker
docker run -v /path/to/config.yaml:/config.yaml -v /path/to/backups:/backups \
  ghcr.io/totallynotrobots/apply-retention-policy:latest prune --config /config.yaml
```

### Command-line Options

- `--config, -c`: Path to configuration file (default: `$HOME/.apply-retention-policy.yaml`)
- `--dry-run, -d`: Show what would be deleted without actually deleting
- `--log-level, -l`: Log level (debug, info, warn, error)

## File Pattern

The file pattern supports the following placeholders:
- `{year}`: 4-digit year (e.g., 2024)
- `{month}`: 2-digit month (01-12)
- `{day}`: 2-digit day (01-31)
- `{hour}`: 2-digit hour (00-23)
- `{minute}`: 2-digit minute (00-59)

Example: `backup-{year}-{month}-{day}-{hour}-{minute}.tar.gz`

## Development

### Prerequisites

- Go 1.21 or later
- Bazel
- Docker (for building container images)

### Building

```bash
# Build binary (using the root target)
bazel build //:apply-retention-policy

# Build multi-arch container image (using the root target)
bazel build //:image
```

### Testing

```bash
bazel test //...
```

## License

MIT License - see LICENSE file for details
