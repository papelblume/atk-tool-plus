package atk

import (
	"fmt"
	"strings"
	"time"

	"github.com/sstallion/go-hid"
)

// BatteryInfo holds the parsed battery percentage, voltage, and charging state.
type BatteryInfo struct {
	Percentage uint8   `json:"percentage"`
	Voltage    float32 `json:"voltage"`
	Charging   bool    `json:"charging"`
}

// DeviceInfo represents details of a discovered ATK peripheral.
type DeviceInfo struct {
	Path        string `json:"path"`
	VendorID    uint16 `json:"vendor_id"`
	ProductID   uint16 `json:"product_id"`
	Interface   int    `json:"interface"`
	UsagePage   uint16 `json:"usage_page"`
	Usage       uint16 `json:"usage"`
	ProductName string `json:"product_name"` // Retrieved from device
	ModelName   string `json:"model_name"`   // Matched from registry
	ReportID    byte   `json:"report_id"`    // Matched from registry
}

// Device represents an open connection to an ATK device.
type Device struct {
	info *DeviceInfo
	dev  *hid.Device
}

// Info returns metadata about the connected device.
func (d *Device) Info() *DeviceInfo {
	return d.info
}

// Close terminates the HID connection to the device.
func (d *Device) Close() error {
	if d.dev != nil {
		return d.dev.Close()
	}
	return nil
}

// QueryBattery queries the device for battery status.
// Dispatches to the appropriate protocol handler based on UsagePage:
//
//	0xFF05            → v2.0 A9 Plus v2.0 / NK protocol
//	0xFF02 / 0xFF04   → v1.0 A9 Plus / Nearlink protocol
func (d *Device) QueryBattery() (*BatteryInfo, error) {
	if d.info.UsagePage == 0xFF05 {
		return d.QueryBatteryV2()
	}
	return d.QueryBatteryV1()
}

// queryBatteryV1 handles v1.0 devices (UsagePage 0xFF02 / 0xFF04):
// ATK A9 Plus (PID 0x1115) and Nearlink Mouse Dongle (PID 0x10c9).
//
// Packet: FinalizePayload-constructed 16-byte frame prepended with ReportID.
// Response layout (big-endian voltage):
//
//	[6]  battery percentage
//	[7]  charging flag (0x01 = charging)
//	[8]  voltage high byte
//	[9]  voltage low byte
func (d *Device) QueryBatteryV1() (*BatteryInfo, error) {
	// Construct raw payload frame (16 bytes)
	payload := FinalizePayload(CmdQueryBattery, [14]byte{})

	// Prepend Report ID to create the transfer packet
	packet := make([]byte, 1+len(payload))
	packet[0] = d.info.ReportID
	copy(packet[1:], payload)

	// Write packet to device
	_, err := d.dev.Write(packet)
	if err != nil {
		return nil, fmt.Errorf("failed to write query packet: %w", err)
	}

	// Read response (expected layout: header/opcode, metadata, data bytes)
	inBuf := make([]byte, 64)
	n, err := d.dev.ReadWithTimeout(inBuf, 200*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("read timeout or error: %w", err)
	}
	if n < 10 {
		return nil, fmt.Errorf("response frame too short (got %d bytes, expected >= 10)", n)
	}

	// Parse out battery percentage (index 6)
	batteryPercent := inBuf[6]

	// Parse out charging state (index 7). 0x01 means charging, 0x00 means discharging.
	charging := inBuf[7] == 0x01

	// Parse out battery voltage in millivolts (indices 8 & 9, big-endian)
	millivolts := (uint16(inBuf[8]) << 8) | uint16(inBuf[9])
	voltage := float32(millivolts) / 1000.0

	return &BatteryInfo{
		Percentage: batteryPercent,
		Voltage:    voltage,
		Charging:   charging,
	}, nil
}

// queryBatteryV2 handles v2.0 devices (UsagePage 0xFF05):
// ATK A9 Plus v2.0 NK (PID 0x1263) and ATK NK NANO dongle (PID 0x1216).
//
// Response layout (little-endian voltage):
//
//	 [0]  report id
//		[1]  CmdMarkerByte, 0x72, included with all command responses
//		[2]  0x00 always
//		[3]  charging flag (0x3b = charging/wired, 0x05 = discharging)
//		[4]  0x00 always
//		[5]  battery reply signature byte (must be 0x07)
//		[6]  battery reply signature byte (must be 0x01)
//		[7]  battery percentage
//		[8]  voltage low byte
//		[9]  voltage high byte
func (d *Device) QueryBatteryV2() (*BatteryInfo, error) {
	packet := make([]byte, FrameSizeV2)
	packet[0] = d.info.ReportID
	packet[1] = 0x00          // transaction byte — doesn't appear to matter what value
	packet[2] = CmdMarkerByte // sent with all commands, not just battery
	packet[3] = 0x00          // transaction byte — doesn't appear to matter what value
	packet[4] = 0x00          // transaction byte — doesn't appear to matter what value
	if strings.Contains(strings.ToLower(d.info.ProductName), "dongle") {
		packet[5] = 0x01 // wireless NANO dongle
	} else {
		packet[5] = 0x00 // wired
	}
	packet[6] = CmdQueryBatteryV2ByteOne // first battery query subcommand byte
	packet[7] = CmdQueryBatteryV2ByteTwo // second battery query subcommand byte
	// remaining bytes 8-63 already zero from make()

	// Write packet to device
	_, err := d.dev.Write(packet)
	if err != nil {
		return nil, fmt.Errorf("failed to write query packet: %w", err)
	}

	// Read response
	inBuf := make([]byte, FrameSizeV2)
	n, err := d.dev.ReadWithTimeout(inBuf, 200*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("read timeout or error: %w", err)
	}
	if n < 10 {
		return nil, fmt.Errorf("response frame too short (got %d bytes, expected >= 10)", n)
	}

	// Parse out battery percentage (index 7)
	batteryPercent := inBuf[7]

	// Parse out charging state (index 3). 0x3b means charging, 0x05 means discharging.
	charging := inBuf[3] == 0x3b

	// Parse out battery voltage in millivolts (indices 8 & 9), Little-endian
	millivolts := (uint16(inBuf[8]) | uint16(inBuf[9])<<8)
	voltage := float32(millivolts) / 1000.0

	return &BatteryInfo{
		Percentage: batteryPercent,
		Charging:   charging,
		Voltage:    voltage,
	}, nil
}
