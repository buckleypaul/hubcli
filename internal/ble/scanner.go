package ble

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/hubblenetwork/hubcli/internal/models"
	"tinygo.org/x/bluetooth"
)

var (
	// ErrScanTimeout indicates the scan timed out without finding packets
	ErrScanTimeout = errors.New("scan timeout")

	// ErrAdapterNotEnabled indicates Bluetooth is not enabled
	ErrAdapterNotEnabled = errors.New("bluetooth adapter not enabled")

	// ErrScanInProgress indicates a scan is already running
	ErrScanInProgress = errors.New("scan already in progress")

	// ErrScanStopped indicates the scan was stopped
	ErrScanStopped = errors.New("scan stopped")
)

// ScanResult represents a single BLE scan result
type ScanResult struct {
	Packet *models.EncryptedPacket
	Raw    RawAdvertisement
	Error  error
}

// ScanOptions configures the scanner behavior
type ScanOptions struct {
	// Timeout is the maximum scan duration (0 = no timeout)
	Timeout time.Duration

	// FilterHubbleOnly filters to only Hubble service advertisements
	FilterHubbleOnly bool

	// Location to attach to captured packets
	Location models.Location

	// MaxPackets limits the number of packets to capture (0 = unlimited)
	MaxPackets int
}

// DefaultScanOptions returns sensible default scan options
func DefaultScanOptions() ScanOptions {
	return ScanOptions{
		Timeout:          30 * time.Second,
		FilterHubbleOnly: true,
		Location: models.Location{
			Fake: true, // Mark as local scan by default
		},
		MaxPackets: 0,
	}
}

// Scanner provides BLE scanning capabilities
type Scanner struct {
	adapter  *bluetooth.Adapter
	mu       sync.Mutex
	scanning bool
	stopCh   chan struct{}
}

// NewScanner creates a new BLE scanner
func NewScanner() (*Scanner, error) {
	adapter := bluetooth.DefaultAdapter
	if err := adapter.Enable(); err != nil {
		return nil, errors.Join(ErrAdapterNotEnabled, err)
	}

	return &Scanner{
		adapter: adapter,
	}, nil
}

// IsScanning returns true if a scan is in progress
func (s *Scanner) IsScanning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.scanning
}

// Stop stops an ongoing scan
func (s *Scanner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.scanning && s.stopCh != nil {
		close(s.stopCh)
		s.stopCh = nil
	}
}

// Scan scans for Hubble BLE advertisements and returns all found packets
func (s *Scanner) Scan(ctx context.Context, opts ScanOptions) ([]models.EncryptedPacket, error) {
	s.mu.Lock()
	if s.scanning {
		s.mu.Unlock()
		return nil, ErrScanInProgress
	}
	s.scanning = true
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.scanning = false
		s.mu.Unlock()
	}()

	var packets []models.EncryptedPacket
	var mu sync.Mutex

	// Set up timeout context
	scanCtx := ctx
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		scanCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Channel to signal scan completion
	done := make(chan error, 1)

	// Start scanning
	go func() {
		err := s.adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
			// Check if we should stop
			select {
			case <-s.stopCh:
				adapter.StopScan()
				return
			case <-scanCtx.Done():
				adapter.StopScan()
				return
			default:
			}

			raw := convertScanResult(result)

			// Apply Hubble filter if enabled
			if opts.FilterHubbleOnly && !ContainsHubbleService(raw) {
				return
			}

			// Parse the advertisement
			packet, err := ParseAdvertisement(raw, opts.Location)
			if err != nil {
				return // Skip non-Hubble packets
			}

			mu.Lock()
			packets = append(packets, *packet)
			count := len(packets)
			mu.Unlock()

			// Check if we've reached max packets
			if opts.MaxPackets > 0 && count >= opts.MaxPackets {
				adapter.StopScan()
			}
		})
		done <- err
	}()

	// Wait for scan to complete or context to cancel
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			return packets, err
		}
	case <-scanCtx.Done():
		s.adapter.StopScan()
		<-done // Wait for scan goroutine to finish
	case <-s.stopCh:
		s.adapter.StopScan()
		<-done
		return packets, ErrScanStopped
	}

	return packets, nil
}

// ScanSingle scans until a single Hubble packet is found
func (s *Scanner) ScanSingle(ctx context.Context, opts ScanOptions) (*models.EncryptedPacket, error) {
	opts.MaxPackets = 1
	packets, err := s.Scan(ctx, opts)
	if err != nil {
		return nil, err
	}

	if len(packets) == 0 {
		return nil, ErrScanTimeout
	}

	return &packets[0], nil
}

// ScanStream returns a channel that streams packets as they are discovered
func (s *Scanner) ScanStream(ctx context.Context, opts ScanOptions) (<-chan ScanResult, error) {
	s.mu.Lock()
	if s.scanning {
		s.mu.Unlock()
		return nil, ErrScanInProgress
	}
	s.scanning = true
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	results := make(chan ScanResult, 100)

	// Set up timeout context
	scanCtx := ctx
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		scanCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
		// Cancel will be called when the goroutine exits
		go func() {
			<-scanCtx.Done()
			cancel()
		}()
	}

	go func() {
		defer func() {
			s.mu.Lock()
			s.scanning = false
			s.mu.Unlock()
			close(results)
		}()

		packetCount := 0

		err := s.adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
			// Check if we should stop
			select {
			case <-s.stopCh:
				adapter.StopScan()
				return
			case <-scanCtx.Done():
				adapter.StopScan()
				return
			default:
			}

			raw := convertScanResult(result)

			// Apply Hubble filter if enabled
			if opts.FilterHubbleOnly && !ContainsHubbleService(raw) {
				return
			}

			// Parse the advertisement
			packet, err := ParseAdvertisement(raw, opts.Location)

			scanResult := ScanResult{
				Raw: raw,
			}

			if err != nil {
				scanResult.Error = err
			} else {
				scanResult.Packet = packet
				packetCount++
			}

			// Send to channel (non-blocking)
			select {
			case results <- scanResult:
			default:
				// Channel full, skip this result
			}

			// Check if we've reached max packets
			if opts.MaxPackets > 0 && packetCount >= opts.MaxPackets {
				adapter.StopScan()
			}
		})

		if err != nil {
			results <- ScanResult{Error: err}
		}
	}()

	return results, nil
}

// convertScanResult converts a bluetooth.ScanResult to our RawAdvertisement type
func convertScanResult(result bluetooth.ScanResult) RawAdvertisement {
	raw := RawAdvertisement{
		LocalName:   result.LocalName(),
		RSSI:        int(result.RSSI),
		Address:     result.Address.String(),
		Timestamp:   time.Now(),
		ServiceData: make(map[string][]byte),
	}

	// Extract manufacturer data - combine all manufacturer data elements
	mfgData := result.ManufacturerData()
	for _, elem := range mfgData {
		// Append company ID (2 bytes, little endian) + data
		raw.ManufacturerData = append(raw.ManufacturerData, byte(elem.CompanyID&0xFF), byte(elem.CompanyID>>8))
		raw.ManufacturerData = append(raw.ManufacturerData, elem.Data...)
	}

	// Get service data - iterate through all service data elements
	serviceDataElements := result.ServiceData()
	for _, elem := range serviceDataElements {
		uuidStr := elem.UUID.String()
		raw.ServiceData[uuidStr] = elem.Data
		raw.ServiceUUIDs = append(raw.ServiceUUIDs, uuidStr)
	}

	return raw
}

// MockScanner is a scanner that can be used for testing without real BLE hardware
type MockScanner struct {
	Packets   []models.EncryptedPacket
	Error     error
	scanning  bool
	mu        sync.Mutex
	callbacks []func(ScanResult)
}

// NewMockScanner creates a mock scanner for testing
func NewMockScanner() *MockScanner {
	return &MockScanner{}
}

// SetPackets sets the packets that will be returned by the mock scanner
func (m *MockScanner) SetPackets(packets []models.EncryptedPacket) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Packets = packets
}

// SetError sets an error that will be returned by the mock scanner
func (m *MockScanner) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Error = err
}

// IsScanning returns whether a mock scan is in progress
func (m *MockScanner) IsScanning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.scanning
}

// Stop stops the mock scan
func (m *MockScanner) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scanning = false
}

// Scan returns the pre-configured packets or error
func (m *MockScanner) Scan(ctx context.Context, opts ScanOptions) ([]models.EncryptedPacket, error) {
	m.mu.Lock()
	if m.Error != nil {
		err := m.Error
		m.mu.Unlock()
		return nil, err
	}
	packets := m.Packets
	m.mu.Unlock()

	// Simulate scan time if timeout is set
	if opts.Timeout > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond): // Short delay for tests
		}
	}

	if opts.MaxPackets > 0 && len(packets) > opts.MaxPackets {
		return packets[:opts.MaxPackets], nil
	}

	return packets, nil
}

// ScanSingle returns the first pre-configured packet
func (m *MockScanner) ScanSingle(ctx context.Context, opts ScanOptions) (*models.EncryptedPacket, error) {
	packets, err := m.Scan(ctx, opts)
	if err != nil {
		return nil, err
	}
	if len(packets) == 0 {
		return nil, ErrScanTimeout
	}
	return &packets[0], nil
}

// ScanStream returns a channel with pre-configured packets
func (m *MockScanner) ScanStream(ctx context.Context, opts ScanOptions) (<-chan ScanResult, error) {
	m.mu.Lock()
	if m.Error != nil {
		err := m.Error
		m.mu.Unlock()
		return nil, err
	}
	packets := m.Packets
	m.scanning = true
	m.mu.Unlock()

	results := make(chan ScanResult, len(packets))

	go func() {
		defer func() {
			m.mu.Lock()
			m.scanning = false
			m.mu.Unlock()
			close(results)
		}()

		for i, p := range packets {
			if opts.MaxPackets > 0 && i >= opts.MaxPackets {
				break
			}

			select {
			case <-ctx.Done():
				return
			default:
				packet := p // Copy to avoid reference issues
				results <- ScanResult{Packet: &packet}
				time.Sleep(10 * time.Millisecond) // Simulate discovery time
			}
		}
	}()

	return results, nil
}
