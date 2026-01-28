# Hubble TUI - Go Terminal UI Implementation Plan

A terminal user interface (TUI) for interacting with the Hubble Network API, written in Go using Bubble Tea. This CLI mirrors the functionality of [pyhubblenetwork](https://github.com/HubbleNetwork/pyhubblenetwork) with a fully navigable terminal interface.

## Overview

### Goals
- Fully navigable terminal UI with keyboard-driven navigation
- Credential entry screen when `HUBBLE_ORG_ID` and `HUBBLE_API_TOKEN` are not set
- macOS Keychain integration for secure credential storage
- Full BLE scanning support using native Go bluetooth libraries
- Complete coverage of pyhubblenetwork's cloud API operations
- Comprehensive test suite with unit, integration, and TUI component tests

### Technology Stack
- **Language**: Go 1.22+
- **TUI Framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Bubbles](https://github.com/charmbracelet/bubbles) + [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- **BLE**: [tinygo-org/bluetooth](https://github.com/tinygo-org/bluetooth)
- **Keychain**: [zalando/go-keyring](https://github.com/zalando/go-keyring)
- **HTTP Client**: Standard library `net/http` with custom client wrapper
- **Testing**: Standard `testing` package + [testify](https://github.com/stretchr/testify) + [httptest](https://pkg.go.dev/net/http/httptest)

---

## Project Structure

```
hubcli/
├── cmd/
│   └── hubcli/
│       └── main.go                 # Application entry point
├── internal/
│   ├── api/                        # Hubble Cloud API client
│   │   ├── client.go               # HTTP client wrapper
│   │   ├── client_test.go
│   │   ├── devices.go              # Device operations
│   │   ├── devices_test.go
│   │   ├── organization.go         # Organization operations
│   │   ├── organization_test.go
│   │   ├── packets.go              # Packet operations
│   │   ├── packets_test.go
│   │   └── errors.go               # API error types
│   ├── ble/                        # Bluetooth Low Energy scanning
│   │   ├── scanner.go              # BLE scanner implementation
│   │   ├── scanner_test.go
│   │   ├── packet.go               # BLE packet parsing
│   │   └── packet_test.go
│   ├── crypto/                     # Cryptographic operations
│   │   ├── decrypt.go              # AES-CTR decryption
│   │   ├── decrypt_test.go
│   │   ├── kdf.go                  # SP800-108 key derivation
│   │   └── kdf_test.go
│   ├── auth/                       # Authentication & credentials
│   │   ├── credentials.go          # Credential management
│   │   ├── credentials_test.go
│   │   ├── keychain.go             # macOS Keychain integration
│   │   └── keychain_test.go
│   ├── models/                     # Data models
│   │   ├── device.go
│   │   ├── packet.go
│   │   ├── location.go
│   │   └── organization.go
│   └── tui/                        # Terminal UI components
│       ├── app.go                  # Main application model
│       ├── app_test.go
│       ├── styles.go               # Lip Gloss styling
│       ├── keys.go                 # Key bindings
│       ├── screens/                # TUI screens
│       │   ├── login.go            # Credential entry screen
│       │   ├── login_test.go
│       │   ├── home.go             # Main menu / dashboard
│       │   ├── home_test.go
│       │   ├── devices.go          # Device list & management
│       │   ├── devices_test.go
│       │   ├── packets.go          # Packet viewer
│       │   ├── packets_test.go
│       │   ├── ble_scan.go         # BLE scanning screen
│       │   ├── ble_scan_test.go
│       │   ├── org_info.go         # Organization info screen
│       │   └── org_info_test.go
│       └── components/             # Reusable UI components
│           ├── table.go            # Data table component
│           ├── form.go             # Form input component
│           ├── spinner.go          # Loading spinner
│           ├── status.go           # Status bar
│           └── help.go             # Help overlay
├── pkg/                            # Public packages (if needed)
│   └── hubble/                     # Exported client library
│       └── client.go
├── testdata/                       # Test fixtures
│   ├── packets.json
│   └── devices.json
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## Core Components

### 1. API Client (`internal/api/`)

#### Client Configuration
```go
type Client struct {
    baseURL    string
    orgID      string
    token      string
    httpClient *http.Client
}

type Environment string

const (
    EnvProduction  Environment = "production"
    EnvStaging     Environment = "staging"
    EnvDevelopment Environment = "development"
)
```

#### API Endpoints to Implement

| Operation | Method | Endpoint | Description |
|-----------|--------|----------|-------------|
| `CheckCredentials` | GET | `/org/{org_id}/check` | Validate API credentials |
| `GetOrganization` | GET | `/org/{org_id}` | Retrieve organization metadata |
| `ListDevices` | GET | `/org/{org_id}/devices` | List all registered devices |
| `RegisterDevice` | POST | `/v2/org/{org_id}/devices` | Register a new device |
| `UpdateDevice` | PATCH | `/org/{org_id}/devices/{device_id}` | Update device name/tags |
| `RetrievePackets` | GET | `/org/{org_id}/packets` | Get decrypted packets |
| `IngestPacket` | POST | `/org/{org_id}/packets` | Upload encrypted packets |

#### Error Handling
```go
type APIError struct {
    StatusCode int
    Message    string
    Details    map[string]interface{}
}

var (
    ErrInvalidCredentials = errors.New("invalid credentials")
    ErrNotFound          = errors.New("resource not found")
    ErrRateLimited       = errors.New("rate limited")
)
```

---

### 2. Data Models (`internal/models/`)

```go
// Device represents a registered Hubble device
type Device struct {
    ID         string            `json:"device_id"`
    Name       string            `json:"name,omitempty"`
    Key        string            `json:"key"`           // Base64-encoded encryption key
    Encryption string            `json:"encryption"`    // "AES-256-CTR" or "AES-128-CTR"
    Tags       map[string]string `json:"tags,omitempty"`
    CreatedAt  time.Time         `json:"created_at"`
}

// Location represents geographic coordinates
type Location struct {
    Latitude           float64   `json:"latitude"`
    Longitude          float64   `json:"longitude"`
    Timestamp          time.Time `json:"timestamp"`
    HorizontalAccuracy float64   `json:"horizontal_accuracy,omitempty"`
    Altitude           float64   `json:"altitude,omitempty"`
    VerticalAccuracy   float64   `json:"vertical_accuracy,omitempty"`
    Fake               bool      `json:"fake,omitempty"`
}

// EncryptedPacket represents a raw BLE advertisement
type EncryptedPacket struct {
    Payload   []byte    `json:"payload"`
    RSSI      int       `json:"rssi"`
    Timestamp time.Time `json:"timestamp"`
    Location  Location  `json:"location"`
}

// DecryptedPacket represents a successfully decrypted packet
type DecryptedPacket struct {
    DeviceID    string    `json:"device_id"`
    Payload     []byte    `json:"payload"`
    TimeCounter uint32    `json:"time_counter"`
    Timestamp   time.Time `json:"timestamp"`
    Location    Location  `json:"location"`
}

// Organization represents org metadata
type Organization struct {
    ID   string `json:"org_id"`
    Name string `json:"name"`
}
```

---

### 3. Cryptographic Operations (`internal/crypto/`)

#### Key Derivation (SP800-108 Counter Mode)
```go
// DeriveKey implements NIST SP 800-108 counter-mode KDF using AES-CMAC
func DeriveKey(key []byte, label, context string, length int) ([]byte, error)

// DeriveNonce generates a 12-byte nonce for AES-CTR
func DeriveNonce(key []byte, counter uint32) ([]byte, error)

// DeriveEncryptionKey generates the encryption key from master key
func DeriveEncryptionKey(key []byte, counter uint32) ([]byte, error)
```

#### Decryption
```go
// Decrypt attempts to decrypt a packet using the provided key
// It searches a time window (default ±2 days) for the correct counter
func Decrypt(key []byte, packet EncryptedPacket, options ...DecryptOption) (*DecryptedPacket, error)

// DecryptOption configures decryption behavior
type DecryptOption func(*decryptConfig)

func WithTimeWindow(days int) DecryptOption
func WithExpectedTime(t time.Time) DecryptOption
```

#### Authentication
```go
// ComputeAuthTag generates a 4-byte CMAC authentication tag
func ComputeAuthTag(key, data []byte) ([]byte, error)

// VerifyAuthTag checks if the auth tag matches
func VerifyAuthTag(key, data, tag []byte) bool
```

---

### 4. BLE Scanner (`internal/ble/`)

```go
const TargetServiceUUID = "0000fca6-0000-1000-8000-00805f9b34fb"

// Scanner handles BLE advertisement scanning
type Scanner struct {
    adapter *bluetooth.Adapter
}

// Scan scans for Hubble BLE advertisements
func (s *Scanner) Scan(ctx context.Context, timeout time.Duration) ([]EncryptedPacket, error)

// ScanSingle captures a single matching advertisement
func (s *Scanner) ScanSingle(ctx context.Context, timeout time.Duration) (*EncryptedPacket, error)

// ScanStream returns a channel of packets for real-time streaming
func (s *Scanner) ScanStream(ctx context.Context) (<-chan EncryptedPacket, error)
```

---

### 5. Credential Management (`internal/auth/`)

```go
const (
    KeychainService = "hubcli"
    OrgIDKey       = "org_id"
    TokenKey       = "api_token"
)

// CredentialStore manages credential persistence
type CredentialStore interface {
    Get() (*Credentials, error)
    Save(creds *Credentials) error
    Delete() error
    Exists() bool
}

// Credentials holds authentication data
type Credentials struct {
    OrgID string
    Token string
}

// KeychainStore implements CredentialStore using macOS Keychain
type KeychainStore struct{}

// EnvStore implements CredentialStore using environment variables (read-only)
type EnvStore struct{}

// GetCredentials returns credentials from keychain or env vars
func GetCredentials() (*Credentials, error)
```

---

## TUI Architecture

### Application State Machine

```
┌─────────────────────────────────────────────────────────────┐
│                        Application                          │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────┐    ┌─────────┐    ┌─────────────────────────┐ │
│  │  Login  │───▶│  Home   │───▶│  Sub-screens            │ │
│  │ Screen  │    │ Screen  │    │  - Devices              │ │
│  └─────────┘    └─────────┘    │  - Packets              │ │
│       │              │         │  - BLE Scan             │ │
│       │              │         │  - Organization Info    │ │
│       ▼              ▼         │  - Register Device      │ │
│  [No Creds]     [Has Creds]    └─────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Screen Definitions

#### Login Screen
- Shown when no credentials exist in keychain or env vars
- Two text inputs: Organization ID and API Token
- "Validate & Save" button to verify credentials before storing
- Error display for invalid credentials
- Option to skip to env var instructions

#### Home Screen (Main Menu)
- Organization name display at top
- Menu options:
  - **Devices** - View and manage devices
  - **Packets** - View packet history
  - **BLE Scan** - Scan for local BLE advertisements
  - **Organization Info** - View org metadata
  - **Settings** - Manage credentials
  - **Quit** - Exit application

#### Devices Screen
- Paginated table of devices with columns: ID, Name, Encryption, Created
- Actions:
  - **[n]** New device - Register a new device
  - **[Enter]** View device details
  - **[r]** Rename device
  - **[p]** View device packets
  - **[/]** Search/filter devices

#### Packets Screen
- Table view with columns: Device ID, Timestamp, Location, Payload (hex)
- Filter by device ID
- Configurable time window (default: 7 days)
- Export to JSON/CSV
- Auto-refresh toggle

#### BLE Scan Screen
- Real-time scanning display
- Packet count and signal strength indicators
- Decryption status (if key provided)
- Options:
  - Set timeout
  - Select device key for decryption
  - Ingest packets to cloud
  - Export raw packets

#### Organization Info Screen
- Display org metadata
- Credential status
- API connectivity check

### Key Bindings

| Key | Action |
|-----|--------|
| `q`, `Ctrl+C` | Quit / Back |
| `?` | Toggle help |
| `Enter` | Select / Confirm |
| `Tab` | Next field |
| `Shift+Tab` | Previous field |
| `↑/↓` or `j/k` | Navigate list |
| `g/G` | Go to top/bottom |
| `/` | Search |
| `r` | Refresh data |
| `Esc` | Cancel / Close modal |

### Styling (Lip Gloss)

```go
var (
    // Colors
    Primary    = lipgloss.Color("#7B68EE")  // Medium slate blue
    Secondary  = lipgloss.Color("#00CED1")  // Dark turquoise
    Success    = lipgloss.Color("#32CD32")  // Lime green
    Warning    = lipgloss.Color("#FFD700")  // Gold
    Error      = lipgloss.Color("#FF6347")  // Tomato
    Subtle     = lipgloss.Color("#666666")  // Gray

    // Base styles
    TitleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(Primary).
        MarginBottom(1)

    SelectedStyle = lipgloss.NewStyle().
        Background(Primary).
        Foreground(lipgloss.Color("#FFFFFF"))

    StatusBarStyle = lipgloss.NewStyle().
        Background(lipgloss.Color("#333333")).
        Foreground(lipgloss.Color("#FFFFFF")).
        Padding(0, 1)
)
```

---

## Testing Strategy

### Unit Tests

#### API Client Tests (`internal/api/*_test.go`)
- Mock HTTP server using `httptest.Server`
- Test all API operations with success and error responses
- Test pagination handling
- Test timeout and retry behavior

```go
func TestListDevices(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "/org/test-org/devices", r.URL.Path)
        assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
        json.NewEncoder(w).Encode([]Device{{ID: "dev-1", Name: "Test"}})
    }))
    defer server.Close()

    client := NewClient(server.URL, "test-org", "test-token")
    devices, err := client.ListDevices(context.Background())

    assert.NoError(t, err)
    assert.Len(t, devices, 1)
}
```

#### Crypto Tests (`internal/crypto/*_test.go`)
- Test vectors from pyhubblenetwork
- Test key derivation with known inputs/outputs
- Test decryption with sample encrypted packets
- Test auth tag computation and verification

#### BLE Tests (`internal/ble/*_test.go`)
- Mock BLE adapter interface for unit testing
- Test packet parsing from raw advertisements
- Test UUID filtering logic
- Test timeout behavior

### Integration Tests

Located in `internal/api/integration_test.go` with build tag `// +build integration`

```go
func TestIntegration_FullWorkflow(t *testing.T) {
    if os.Getenv("HUBBLE_ORG_ID") == "" {
        t.Skip("Integration tests require HUBBLE_ORG_ID")
    }

    client := NewClientFromEnv()

    // Test credential validation
    err := client.CheckCredentials(context.Background())
    require.NoError(t, err)

    // Test device registration
    device, err := client.RegisterDevice(context.Background(), RegisterDeviceRequest{
        Encryption: "AES-256-CTR",
    })
    require.NoError(t, err)

    // Test device listing
    devices, err := client.ListDevices(context.Background())
    require.NoError(t, err)
    assert.Contains(t, deviceIDs(devices), device.ID)

    // Cleanup...
}
```

### TUI Component Tests

Using Bubble Tea's testing utilities:

```go
func TestLoginScreen_Submit(t *testing.T) {
    m := NewLoginModel()

    // Simulate typing org ID
    m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("test-org")})

    // Tab to next field
    m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})

    // Type token
    m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("test-token")})

    // Submit
    m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

    // Verify command was issued
    assert.NotNil(t, cmd)
}
```

### Test Coverage Goals
- API client: >90% coverage
- Crypto operations: 100% coverage (security critical)
- BLE parsing: >85% coverage
- TUI screens: >75% coverage

---

## Build & Deployment

### Makefile Targets

```makefile
.PHONY: build test lint run clean

# Build the binary
build:
	go build -o bin/hubcli ./cmd/hubcli

# Run all tests
test:
	go test -v -race ./...

# Run integration tests (requires credentials)
test-integration:
	go test -v -tags=integration ./internal/api/...

# Run with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Lint
lint:
	golangci-lint run

# Run the TUI
run:
	go run ./cmd/hubcli

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html
```

### Build Tags

- `integration` - Integration tests requiring real API credentials
- `darwin` - macOS-specific code (Keychain, CoreBluetooth)

### Dependencies

```go
// go.mod
module github.com/hubblenetwork/hubcli

go 1.22

require (
    github.com/charmbracelet/bubbletea v1.1.0
    github.com/charmbracelet/bubbles v0.20.0
    github.com/charmbracelet/lipgloss v0.13.0
    github.com/zalando/go-keyring v0.2.5
    github.com/stretchr/testify v1.9.0
    tinygo.org/x/bluetooth v0.10.0
)
```

---

## Implementation Phases

### Phase 1: Core Infrastructure
1. Set up project structure and Go modules
2. Implement data models
3. Implement API client with all endpoints
4. Write unit tests for API client
5. Implement credential management (env + keychain)

### Phase 2: Cryptography
1. Implement SP800-108 key derivation
2. Implement AES-CTR decryption
3. Implement CMAC authentication
4. Add comprehensive test vectors
5. Verify against pyhubblenetwork test cases

### Phase 3: TUI Framework
1. Set up Bubble Tea application structure
2. Implement Login screen
3. Implement Home screen with navigation
4. Implement basic styling system
5. Add key binding infrastructure

### Phase 4: Feature Screens
1. Implement Devices screen with table view
2. Implement Packets screen
3. Implement Organization Info screen
4. Add device registration flow
5. Add search/filter functionality

### Phase 5: BLE Integration
1. Set up tinygo-org/bluetooth adapter
2. Implement BLE scanner
3. Implement BLE Scan screen
4. Add packet ingestion workflow
5. Add real-time decryption display

### Phase 6: Polish & Testing
1. Complete integration test suite
2. Complete TUI component tests
3. Add error handling and edge cases
4. Performance optimization
5. Documentation and README

---

## API Reference (Quick Reference)

### Base URLs
- Production: `https://api.hubblenetwork.com`
- Staging: `https://api.staging.hubblenetwork.com`

### Authentication
All requests require `Authorization: Bearer {token}` header.

### Common Response Codes
- `200` - Success
- `400` - Bad request
- `401` - Invalid credentials
- `404` - Resource not found
- `429` - Rate limited
- `500` - Server error

---

## Open Questions / Future Considerations

1. **Pagination Strategy**: How many items per page? Virtual scrolling for large lists?
2. **Offline Support**: Cache device list locally for offline viewing?
3. **Multi-org Support**: Allow switching between organizations?
4. **Export Formats**: CSV, JSON, or both for packet export?
5. **Notifications**: Alert when BLE scan finds packets?
6. **Themes**: Light/dark mode toggle?
