package atk

import (
	"fmt"

	"github.com/sstallion/go-hid"
)

// Enumerate scans the USB bus for registered and supported ATK devices.
// It enumerates once per unique vendor ID across the registry rather than
// once per device entry, keeping HID subsystem calls constant regardless
// of registry size.
func Enumerate() ([]*DeviceInfo, error) {
	var found []*DeviceInfo
	seenPaths := make(map[string]bool)

	// Collect unique vendor IDs to minimise enumerate calls.
	vendorIDs := make(map[uint16]bool)
	for _, def := range RegisteredDevices() {
		vendorIDs[def.VendorID] = true
	}

	for vid := range vendorIDs {
		err := hid.Enumerate(vid, 0, func(info *hid.DeviceInfo) error {
			// Discard immediately if not in the registry or wrong usage page.
			def, ok := FindDefinition(info.VendorID, info.ProductID)
			if !ok || !def.Matches(info.UsagePage) {
				return nil
			}
			if seenPaths[info.Path] {
				return nil
			}
			seenPaths[info.Path] = true

			productName := info.ProductStr
			if productName == "" {
				productName = "ATK Peripheral"
			}

			found = append(found, &DeviceInfo{
				Path:        info.Path,
				VendorID:    info.VendorID,
				ProductID:   info.ProductID,
				Interface:   info.InterfaceNbr,
				UsagePage:   info.UsagePage,
				Usage:       info.Usage,
				ProductName: productName,
				ModelName:   def.Name,
				ReportID:    def.ReportID,
			})
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to enumerate devices for VID 0x%04x: %w", vid, err)
		}
	}

	return found, nil
}

// Open establishes a connection to a specific ATK device using its DeviceInfo path.
func Open(info *DeviceInfo) (*Device, error) {
	dev, err := hid.OpenPath(info.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open HID path %s: %w", info.Path, err)
	}

	return &Device{
		info: info,
		dev:  dev,
	}, nil
}