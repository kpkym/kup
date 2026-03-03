# kup

Encrypted backup CLI combining [restic](https://restic.net) (encryption & deduplication) and [rclone](https://rclone.org) (cloud transport). Profiles group backup paths with their destination repos so a single command backs up to multiple remotes.

## Prerequisites

- `restic` in `PATH`
- `rclone` in `PATH` (configured with your remotes)

## Installation

```bash
go install github.com/kpkym/kup@latest
```

Or build from source:

```bash
make build   # output: dist/kup
# or
go install .
```

## Configuration

Set `KUP_CONFIG_DIR` to a directory containing `config.toml`:

```bash
export KUP_CONFIG_DIR=~/.config/kup
```

**`config.toml`**

```toml
[global]
rclone_config     = "~/.config/rclone/rclone.conf"
restic_password   = "your-strong-password"
restic_pack_size  = 128          # optional, MB
restic_exclude_file = "~/.config/kup/excludes"  # optional

[profiles.home]
repos = [
  "rclone:backblaze:my-bucket/home",
  "rclone:s3:my-bucket/home",
]
paths = [
  "~/Documents",
  "~/Pictures",
]
```

Profiles can also be split into separate files at `$KUP_CONFIG_DIR/profiles/<name>.toml`:

```toml
# $KUP_CONFIG_DIR/profiles/home.toml
repos = ["rclone:backblaze:my-bucket/home"]
paths = ["~/Documents", "~/Pictures"]
```

## Usage

```bash
# Initialize repos for a profile (run once)
kup init <profile>

# Run a backup
kup backup <profile>
kup backup <profile> --dry-run

# List snapshots
kup snapshots <profile>

# Remove old snapshots
kup forget <profile> --keep-last 10
kup forget <profile> --keep-last 10 --prune

# Remove unreferenced data
kup prune <profile>

# Check repository integrity
kup check <profile>
kup check <profile> --read-data   # full data verification (slow)

# Restore a snapshot
kup restore <snapshot-id> --repo rclone:backblaze:my-bucket/home --target /tmp/restore

# Passthrough commands (inherits kup's environment)
kup restic --repo rclone:backblaze:my-bucket/home -- snapshots
kup rclone -- ls backblaze:my-bucket
```

The `<profile>` argument also accepts a direct repo URI (any string containing `:`), bypassing profile lookup.

## Global Flags

| Flag | Description |
|------|-------------|
| `--config <file>` | Override config file path |
| `--dry-run` | Pass `--dry-run` to restic |
