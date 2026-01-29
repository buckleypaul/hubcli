# Hubble CLI

A terminal user interface (TUI) for interacting with the Hubble Network API, written in Go using [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

- **Fully navigable terminal UI** with keyboard-driven navigation
- **Device Management**: View, register, and manage Hubble devices
- **Packet History**: View and filter packet data from your devices
- **BLE Scanning**: Scan for local Hubble BLE advertisements
- **Organization Info**: View organization details and validate credentials
- **Secure Credentials**: macOS Keychain integration for credential storage

## Installation

### Homebrew (Recommended)

```bash
brew tap buckleypaul/tap
brew install hubcli
```

To upgrade to the latest version:

```bash
brew update && brew upgrade hubcli
```

### From Source

```bash
# Clone the repository
git clone https://github.com/buckleypaul/hubcli.git
cd hubcli

# Build
make build

# Or install to your GOPATH
make install
```

### Requirements

- macOS (for Keychain and BLE support)
- Go 1.22 or later (only if building from source)

## Usage

### Running the CLI

```bash
# Run the built binary
./bin/hubcli

# Or if installed
hubcli
```

### Authentication

The CLI supports two methods of authentication:

#### 1. Environment Variables (Recommended for CI/scripts)

```bash
export HUBBLE_ORG_ID="your-organization-id"
export HUBBLE_API_TOKEN="your-api-token"
```

#### 2. Interactive Login

When no credentials are found, the CLI will display a login screen where you can enter your organization ID and API token. Credentials are securely stored in the macOS Keychain.

### Navigation

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate up/down |
| `Enter` | Select / Confirm |
| `Tab` | Next field |
| `Shift+Tab` | Previous field |
| `Esc` | Back / Cancel |
| `q` | Quit |
| `?` | Toggle help |
| `r` | Refresh data |

### Screens

#### Home Menu
The main menu provides access to all features:
- **Devices** - View and manage registered devices
- **Packets** - View packet history
- **BLE Scan** - Scan for BLE advertisements
- **Organization** - View org info and validate credentials
- **Settings** - Manage stored credentials

#### Devices Screen
- View all registered devices in a table format
- Press `n` to register a new device
- Press `Enter` to view packets for selected device

#### Packets Screen
- View packet history with device ID, timestamp, location, and payload
- Filter by device (press `c` to clear filter)
- Change time window: `1` (1 day), `7` (7 days), `Alt+3` (30 days)

#### BLE Scan Screen
- Scanning starts automatically when entering the screen
- Press `p` or `Space` to pause/resume scanning
- Press `c` to clear captured packets
- Press `Esc` to return to home

#### Settings Screen
- View credential status (Keychain vs Environment)
- Press `c` to clear stored keychain credentials

## Development

### Project Structure

```
hubcli/
├── cmd/hubcli/          # Application entry point
├── internal/
│   ├── api/             # Hubble Cloud API client
│   ├── auth/            # Credential management
│   ├── ble/             # BLE scanning
│   ├── crypto/          # Cryptographic operations
│   ├── models/          # Data models
│   └── tui/             # Terminal UI
│       ├── common/      # Shared styles and keys
│       └── screens/     # Individual screens
├── Makefile
└── plan.md              # Implementation plan
```

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run in development mode (no build)
make dev
```

### Testing

```bash
# Run all tests
make test

# Run with coverage report
make test-coverage

# Run integration tests (requires credentials)
export HUBBLE_ORG_ID="your-org-id"
export HUBBLE_API_TOKEN="your-token"
make test-integration
```

### Code Quality

```bash
# Format code
make fmt

# Run linter (requires golangci-lint)
make lint

# Run all checks
make check
```

## API Reference

The CLI interacts with the Hubble Network API:

| Operation | Description |
|-----------|-------------|
| `CheckCredentials` | Validate API credentials |
| `GetOrganization` | Get organization metadata |
| `ListDevices` | List all registered devices |
| `RegisterDevice` | Register a new device |
| `UpdateDevice` | Update device name/tags |
| `RetrievePackets` | Get decrypted packets |
| `IngestPacket` | Upload encrypted packets |

## Cryptography

The CLI implements Hubble's encryption scheme:

- **Key Derivation**: NIST SP 800-108 Counter Mode KDF with AES-CMAC
- **Encryption**: AES-256-CTR or AES-128-CTR
- **Authentication**: CMAC with 4-byte truncated tags

## BLE Scanning

The BLE scanner looks for advertisements with:
- Service UUID: `0xFCA6` (Hubble Network service)

Scanned packets can be uploaded to the Hubble cloud for processing.

## License

See LICENSE file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `make test`
5. Submit a pull request

## Support

For issues and questions:
- GitHub Issues: [buckleypaul/hubcli](https://github.com/buckleypaul/hubcli/issues)
