# envconfig-docs

A command-line tool that generates markdown documentation for Go configuration structures annotated with `envconfig` tags.

## Installation

```bash
go install github.com/wreulicke/envconfig-docs@latest
```

## Usage

```bash
envconfig-docs <package-path>
```

### Example

```bash
# Generate documentation for current directory
envconfig-docs .

# Generate documentation for a specific package
envconfig-docs ./pkg/config
```

## Features

- Automatically scans Go source files for structs with `envconfig` tags
- Generates markdown tables with configuration details
- Includes information about:
  - Environment variable names
  - Field types
  - Required/optional status
  - Default values
  - Field comments

## Example Output

Given a Go struct like:

```go
type TestConfig struct {
    PGDatabase string `envconfig:"PGDATABASE" required:"true"`
    PGHost     string `envconfig:"PGHOST" default:"localhost"`
    PGPort     int    `envconfig:"PGPORT" default:"15432"`
    PGUser     string `envconfig:"PGUSER" required:"true"`
    PGPassword string `envconfig:"PGPASSWORD" required:"true"`
}
```

The tool generates:

| Name       | Type   | Required | Default     | Comment |
|:-----------|:-------|:---------|:------------|:--------|
| PGDATABASE | string | true     |             |         |
| PGHOST     | string | false    | "localhost" |         |
| PGPORT     | int    | false    | "15432"     |         |
| PGUSER     | string | true     |             |         |
| PGPASSWORD | string | true     |             |         |

## Development

### Prerequisites

- Go 1.23.3 or later

### Build

```bash
go build .
```

### Test

```bash
go test ./...
```

## License

MIT License