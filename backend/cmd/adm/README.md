# Admin CLI Tool

A comprehensive command-line administration tool for the quiz application.

## Quick Start

```bash
# Build the tool
task build-adm

# Show help
task adm-help

# List all users
task run-adm ADM_ARGS="user list"

# Reset a user's password
task run-adm ADM_ARGS="user reset-password username newpassword"

# Show database statistics
task run-adm ADM_ARGS="db stats"
```

## Usage

```bash
./adm [command] [subcommand] [options]
```

### Available Commands

#### User Management (`user`)

- `list` - List all users in the database
- `reset-password [username] [new-password]` - Reset password for a specific user

#### Database Management (`db`)

- `stats` - Show database statistics
- `cleanup` - Run database cleanup operations

## Examples

```bash
# List all users
./adm user list

# Reset password for user "john"
./adm user reset-password john newpassword123

# Show database statistics
./adm db stats

# Run database cleanup
./adm db cleanup
```

## Taskfile Commands

```bash
# Build and run locally
task build-adm
task run-adm ADM_ARGS="user list"

# Show help
task adm-help

# Run specific commands
task run-adm ADM_ARGS="user reset-password admin newpass"
task run-adm ADM_ARGS="db stats"
```

## Architecture

The admin CLI uses a modular command structure:

```
cmd/adm/
├── main.go           # Main entry point
└── commands/         # Command modules
    ├── user.go       # User management commands
    └── db.go         # Database management commands
```

### Adding New Commands

To add new commands:

1. Create a new file in `commands/` (e.g., `commands/system.go`)
2. Define your command functions following the pattern in existing files
3. Add the command to `main.go` in the `main()` function

Example:

```go
// commands/system.go
package commands

import (
    "github.com/spf13/cobra"
)

func SystemCommands() *cobra.Command {
    systemCmd := &cobra.Command{
        Use:   "system",
        Short: "System management commands",
    }

    systemCmd.AddCommand(restartCmd())
    return systemCmd
}

func restartCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "restart",
        Short: "Restart system services",
        RunE:  runRestart,
    }
}

func runRestart(cmd *cobra.Command, args []string) error {
    // Implementation here
    return nil
}
```

Then add to `main.go`:

```go
rootCmd.AddCommand(commands.SystemCommands())
```

## Notes

- Uses the same configuration system as other CLI tools
- Requires database access (local or Docker network)
- All operations are logged with structured logging
- Supports the same environment variables as other tools
- Gracefully handles database connection issues
