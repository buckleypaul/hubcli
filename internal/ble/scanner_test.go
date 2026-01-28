package ble

import (
	"context"
	"testing"
	"time"

	"github.com/hubblenetwork/hubcli/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestDefaultScanOptions(t *testing.T) {
	opts := DefaultScanOptions()

	assert.Equal(t, 30*time.Second, opts.Timeout)
	assert.True(t, opts.FilterHubbleOnly)
	assert.True(t, opts.Location.Fake)
	assert.Equal(t, 0, opts.MaxPackets)
}

func TestMockScanner_NewMockScanner(t *testing.T) {
	scanner := NewMockScanner()
	assert.NotNil(t, scanner)
	assert.Empty(t, scanner.Packets)
	assert.Nil(t, scanner.Error)
}

func TestMockScanner_SetPackets(t *testing.T) {
	scanner := NewMockScanner()

	packets := []models.EncryptedPacket{
		{
			Payload:   []byte{0x01, 0x02, 0x03},
			RSSI:      -65,
			Timestamp: time.Now(),
		},
	}

	scanner.SetPackets(packets)
	assert.Equal(t, packets, scanner.Packets)
}

func TestMockScanner_SetError(t *testing.T) {
	scanner := NewMockScanner()
	err := assert.AnError

	scanner.SetError(err)
	assert.Equal(t, err, scanner.Error)
}

func TestMockScanner_Scan(t *testing.T) {
	scanner := NewMockScanner()

	packets := []models.EncryptedPacket{
		{Payload: []byte{0x01}, RSSI: -60},
		{Payload: []byte{0x02}, RSSI: -70},
	}
	scanner.SetPackets(packets)

	ctx := context.Background()
	opts := DefaultScanOptions()

	result, err := scanner.Scan(ctx, opts)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, packets[0].Payload, result[0].Payload)
	assert.Equal(t, packets[1].Payload, result[1].Payload)
}

func TestMockScanner_Scan_WithMaxPackets(t *testing.T) {
	scanner := NewMockScanner()

	packets := []models.EncryptedPacket{
		{Payload: []byte{0x01}, RSSI: -60},
		{Payload: []byte{0x02}, RSSI: -70},
		{Payload: []byte{0x03}, RSSI: -80},
	}
	scanner.SetPackets(packets)

	ctx := context.Background()
	opts := DefaultScanOptions()
	opts.MaxPackets = 2

	result, err := scanner.Scan(ctx, opts)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestMockScanner_Scan_WithError(t *testing.T) {
	scanner := NewMockScanner()
	scanner.SetError(assert.AnError)

	ctx := context.Background()
	opts := DefaultScanOptions()

	result, err := scanner.Scan(ctx, opts)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestMockScanner_Scan_ContextCanceled(t *testing.T) {
	scanner := NewMockScanner()
	scanner.SetPackets([]models.EncryptedPacket{{Payload: []byte{0x01}}})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	opts := DefaultScanOptions()
	opts.Timeout = 5 * time.Second

	_, err := scanner.Scan(ctx, opts)

	assert.ErrorIs(t, err, context.Canceled)
}

func TestMockScanner_ScanSingle(t *testing.T) {
	scanner := NewMockScanner()

	packets := []models.EncryptedPacket{
		{Payload: []byte{0x01}, RSSI: -60},
		{Payload: []byte{0x02}, RSSI: -70},
	}
	scanner.SetPackets(packets)

	ctx := context.Background()
	opts := DefaultScanOptions()

	result, err := scanner.ScanSingle(ctx, opts)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, packets[0].Payload, result.Payload)
}

func TestMockScanner_ScanSingle_NoPackets(t *testing.T) {
	scanner := NewMockScanner()
	scanner.SetPackets([]models.EncryptedPacket{})

	ctx := context.Background()
	opts := DefaultScanOptions()

	result, err := scanner.ScanSingle(ctx, opts)

	assert.ErrorIs(t, err, ErrScanTimeout)
	assert.Nil(t, result)
}

func TestMockScanner_ScanStream(t *testing.T) {
	scanner := NewMockScanner()

	packets := []models.EncryptedPacket{
		{Payload: []byte{0x01}, RSSI: -60},
		{Payload: []byte{0x02}, RSSI: -70},
	}
	scanner.SetPackets(packets)

	ctx := context.Background()
	opts := DefaultScanOptions()

	resultCh, err := scanner.ScanStream(ctx, opts)

	assert.NoError(t, err)
	assert.NotNil(t, resultCh)

	var results []ScanResult
	for result := range resultCh {
		results = append(results, result)
	}

	assert.Len(t, results, 2)
	assert.NotNil(t, results[0].Packet)
	assert.NotNil(t, results[1].Packet)
}

func TestMockScanner_ScanStream_WithMaxPackets(t *testing.T) {
	scanner := NewMockScanner()

	packets := []models.EncryptedPacket{
		{Payload: []byte{0x01}},
		{Payload: []byte{0x02}},
		{Payload: []byte{0x03}},
	}
	scanner.SetPackets(packets)

	ctx := context.Background()
	opts := DefaultScanOptions()
	opts.MaxPackets = 2

	resultCh, err := scanner.ScanStream(ctx, opts)

	assert.NoError(t, err)

	var results []ScanResult
	for result := range resultCh {
		results = append(results, result)
	}

	assert.Len(t, results, 2)
}

func TestMockScanner_ScanStream_WithError(t *testing.T) {
	scanner := NewMockScanner()
	scanner.SetError(assert.AnError)

	ctx := context.Background()
	opts := DefaultScanOptions()

	resultCh, err := scanner.ScanStream(ctx, opts)

	assert.Error(t, err)
	assert.Nil(t, resultCh)
}

func TestMockScanner_IsScanning(t *testing.T) {
	scanner := NewMockScanner()

	assert.False(t, scanner.IsScanning())

	// Start a stream scan
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	packets := []models.EncryptedPacket{
		{Payload: []byte{0x01}},
	}
	scanner.SetPackets(packets)

	resultCh, _ := scanner.ScanStream(ctx, DefaultScanOptions())

	// IsScanning should be true briefly
	// Note: Due to goroutine timing, this might not always be true
	// so we just read from the channel to let the scan complete

	for range resultCh {
		// Drain channel
	}

	// After scan completes, should be false
	assert.False(t, scanner.IsScanning())
}

func TestMockScanner_Stop(t *testing.T) {
	scanner := NewMockScanner()

	// Set scanning to true manually
	scanner.mu.Lock()
	scanner.scanning = true
	scanner.mu.Unlock()

	assert.True(t, scanner.IsScanning())

	scanner.Stop()

	assert.False(t, scanner.IsScanning())
}

func TestScanResult_Fields(t *testing.T) {
	packet := &models.EncryptedPacket{
		Payload: []byte{0x01, 0x02},
		RSSI:    -65,
	}

	raw := RawAdvertisement{
		Address: "AA:BB:CC:DD:EE:FF",
		RSSI:    -65,
	}

	result := ScanResult{
		Packet: packet,
		Raw:    raw,
		Error:  nil,
	}

	assert.NotNil(t, result.Packet)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", result.Raw.Address)
	assert.Nil(t, result.Error)
}

func TestScanResult_WithError(t *testing.T) {
	result := ScanResult{
		Error: ErrNotHubblePacket,
	}

	assert.Nil(t, result.Packet)
	assert.ErrorIs(t, result.Error, ErrNotHubblePacket)
}

func TestErrors(t *testing.T) {
	// Verify error values are distinct
	errors := []error{
		ErrScanTimeout,
		ErrAdapterNotEnabled,
		ErrScanInProgress,
		ErrScanStopped,
		ErrInvalidPayload,
		ErrPayloadTooShort,
		ErrNotHubblePacket,
	}

	seen := make(map[string]bool)
	for _, err := range errors {
		msg := err.Error()
		assert.False(t, seen[msg], "duplicate error message: %s", msg)
		seen[msg] = true
	}
}
