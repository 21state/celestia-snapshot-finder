# Celestia Snapshot Finder

[![GitHub release (latest by date including pre-releases)](https://img.shields.io/github/v/release/21state/celestia-snapshot-finder)](https://github.com/21state/celestia-snapshot-finder/releases/latest)

CLI tool for downloading Celestia node snapshots. Uses [celestia-snapshots](https://github.com/21state/celestia-snapshots) repository as a source of available snapshots.

## Features

- Automatic selection of fastest snapshot provider
- Progress tracking and resume capability
- Support for consensus/bridge nodes and pruned/archive snapshots

## Installation

```bash
# Build from source
git clone https://github.com/21state/celestia-snapshot-finder.git
cd celestia-snapshot-finder
./build.sh

# Or download pre-built binary from Releases
```

## Usage

```bash
# Download pruned consensus node snapshot
celestia-snapshot-finder consensus pruned

# Download archive bridge node snapshot with manual selection
celestia-snapshot-finder bridge archive --manual

# Additional flags
--chain-id string   Chain ID (default "celestia")
--manual            Enable manual provider selection
```
