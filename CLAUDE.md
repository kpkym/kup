# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`kup` is a CLI backup tool that orchestrates **restic** (encryption/deduplication) and **rclone** (cloud transport) using profile-based configuration. It is a thin wrapper — most logic is delegating to restic/rclone with the right environment variables set.

## Commands

```bash
# Build and install
go install .                          # install to $GOPATH/bin
make build                            # build via goreleaser (output: dist/kup)
make release                          # release via goreleaser (requires git tag)

# Test
go test ./...                         # run all tests
go test ./cmd/...                     # test a specific package

# Lint / vet
go vet ./...
```

The `make build` target also copies the binary to `$CUSTOM_USER_BIN_DIR/kup` if that env var is set.

## Architecture

```
main.go                   → calls cmd.Execute()
cmd/root.go               → cobra root command, config loading, --config/--dry-run flags
cmd/<subcommand>.go       → one file per subcommand registered via init()
internal/config/config.go → Config/GlobalConfig/ProfileConfig structs + Validate()
internal/runner/runner.go → RunRestic, RunRclone, RunResticForEachRepo helpers
```

### Config Loading (`cmd/root.go`)

- Config file: `$KUP_CONFIG_DIR/config.toml` (or `--config` flag)
- Per-profile overrides: `$KUP_CONFIG_DIR/profiles/<name>.toml` — merged into `config.profiles.<name>`
- `~` in `rclone_config` and `restic_exclude_file` paths is expanded to `$HOME`
- Required fields: `global.rclone_config`, `global.restic_password`

### Runner (`internal/runner/runner.go`)

- `RunRestic` / `RunRclone` — execute the binary with env vars set from `GlobalConfig`
- `RunResticForEachRepo` — loops over a profile's repos sequentially, prints `=== <repo> ===` headers, accumulates failures
- SIGINT/SIGTERM are forwarded to child processes

### Subcommands

| Command | Profile/repo arg | Notes |
|---------|-----------------|-------|
| `backup <profile>` | profile name | writes paths to a temp `--files-from` file |
| `forget <profile>` | profile name or `rclone:remote:path` URI | requires `--keep-last` |
| `check <profile>` | profile name or URI | optional `--read-data` |
| `snapshots <profile>` | profile name or URI | |
| `prune <profile>` | profile name or URI | |
| `restore <snapshot-id>` | `--repo` flag | requires `--target` |
| `init-repo <profile>` | profile name or URI | |
| `restic --repo <uri> -- <args>` | `--repo` flag | passthrough, DisableFlagParsing |
| `rclone -- <args>` | n/a | passthrough, DisableFlagParsing |
| `version` | n/a | prints Version/Commit/Date (ldflags) |

`GetRepos()` resolves a profile name to its `repos` list, or treats any string containing `:` as a direct repo URI.

### Adding a New Subcommand

1. Create `cmd/<name>.go` in package `cmd`
2. Define a `var <name>Cmd = &cobra.Command{...}`
3. Register in `func init() { rootCmd.AddCommand(<name>Cmd) }`
4. Use `cfg.GetRepos(args[0])` or `cfg.GetProfile(args[0])` to resolve the profile
5. Delegate execution to `runner.RunRestic`, `runner.RunRclone`, or `runner.RunResticForEachRepo`
