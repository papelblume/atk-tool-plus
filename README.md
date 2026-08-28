# ATK Peripheral Tool

[![Go Reference](https://pkg.go.dev/badge/github.com/noosxe/atk-tool.svg)](https://pkg.go.dev/github.com/noosxe/atk-tool)

`atk-tool-plus` is an extended/enhanced fork of [atk-tool](https://github.com/noosxe/atk-tool), a modular Golang library and command-line utility for querying and interfacing with **ATK gaming peripherals** (focusing on mouse telemetry like battery status and voltage).

This fork adds support for ATK's second protocol generation (usage page `0xFF05`), expands the device registry to 251 entries, and supports querying multiple connected mice at once instead of stopping at the first one found.

It is structured both as a reusable Go library that can be integrated into other projects and as a standalone CLI tool.

---

## Features

- **Telemetry Queries:** Fetch battery charge percentage, current operating voltage, and charging status (wired vs. wireless).
- **Dual Protocol Support:** Handles both the original ATK protocol (UsagePage `0xFF02`/`0xFF04`) and the newer ATK protocol (UsagePage `0xFF05`) transparently, dispatching automatically based on the connected device.
- **Deduplication:** Merges multiple HID interface/endpoint paths into single logical devices.
- **JSON Outputs:** Native structured JSON output formats for seamless scripting and dashboard integration.
- **Highly Modular:** Centralized device registry for easy support expansion of additional models. The current 251 entries *should* contain all ATK and VXE mice variants as of 2027-08.
- **Efficient Enumeration:** Performs a single HID bus scan per unique vendor ID. Device discovery time stays constant regardless of how many devices are in the registry.
- **Subcommand Aliases:** Fast, intuitive CLI commands powered by Cobra (e.g. `atk-tool-plus battery`).

---

## Supported Devices

The registry contains **251 entries** spanning two protocol families:

| Protocol | Usage Pages | Entries |
|---|---|---|
| v1.0 | `0xFF02`, `0xFF04` | 187 |
| v2.0 | `0xFF05` | 64 |

The correct protocol is selected automatically based on the UsagePage exposed by the device — no manual configuration is required.

Registry entries are sourced from the ATK HUB webHID. The following **6 devices have been tested**:

| Device | VID:PID | Connection |
|---|---|---|
| ATK A9 Plus | `373b:1115` | Wired |
| Nearlink Mouse Dongle | `373b:10c9` | Wireless receiver |
| ATK A9 Plus 2.0 / ATK Dragonfly A9 Plus 2.0 | `373b:1263` | Wired |
| NK mouse NANO dongle | `373b:1216` | Wireless receiver |
| ATK VXE R1 Pro | `373b:1272` | Wired |
| VXE NordicMouse 1K Dongle | `3554:f58a` | Wireless receiver |

Untested registry entries may work correctly — if you verify a device not listed above, a note in the issue tracker is welcome.

You can check the `vendor:product` ID of your own devices with `lsusb` (ATK uses vendor IDs `373b` or `3554`).

To request support for more devices, or to add them yourself, see the [Extending Supported Devices](#extending-supported-devices) section below.

---

## Linux Setup (udev rules)

Because this tool interacts with low-level USB HID interfaces (`hidraw`), your Linux user needs read/write permissions for the target device files. Create a udev rule to run the tool without `sudo`:

1. Create a file `/etc/udev/rules.d/99-atk.rules` with the following content:
   ```udev
   # ATK / Compx USB Peripherals & Receiver Dongles
   SUBSYSTEM=="hidraw", ATTRS{idVendor}=="373b", MODE="0666"
   SUBSYSTEM=="hidraw", ATTRS{idVendor}=="3554", MODE="0666"
   ```
2. Reload and trigger the rule configuration:
   ```bash
   sudo udevadm control --reload-rules && sudo udevadm trigger
   ```

---

## Command Line Interface (CLI)

### Installation

To install the latest version of the CLI utility directly into your `$GOPATH/bin`:

```bash
git clone https://github.com/papelblume/atk-tool-plus
go build -o atk-tool-plus ./cmd/atk-tool-plus
```

### Usage

Run the utility without any arguments (or with `--help`) to view the standard help:

```bash
atk-tool-plus
```

#### 1. List connected ATK devices
```bash
atk-tool-plus list
```

#### 2. Get battery telemetry
```bash
atk-tool-plus status
# OR using the alias:
atk-tool-plus battery
```

#### 3. Output in JSON format
```bash
atk-tool-plus status --json
```

#### 4. Target a specific device path (useful when multiple dongles are plugged in)
```bash
atk-tool-plus status --device /dev/hidraw5
```

### Shell Completions

`atk-tool-plus` supports generating autocompletion scripts for Bash, Zsh, Fish, and PowerShell.

#### Bash

This script depends on the `bash-completion` package. If it is not installed already, install it via your OS package manager.

To load completions in your current shell session:
```bash
source <(atk-tool-plus completion bash)
```

To load completions for every new session, execute once:
*   **Linux**:
    ```bash
    atk-tool-plus completion bash > /etc/bash_completion.d/atk-tool
    ```
*   **macOS**:
    ```bash
    atk-tool-plus completion bash > $(brew --prefix)/etc/bash_completion.d/atk-tool
    ```

#### Zsh

If shell completion is not already enabled in your environment, you will need to enable it. You can execute the following once:
```zsh
echo "autoload -U compinit; compinit" >> ~/.zshrc
```

To load completions in your current shell session:
```zsh
source <(atk-tool-plus completion zsh)
```

To load completions for every new session, execute once:
*   **Linux**:
    ```zsh
    atk-tool-plus completion zsh > "${fpath[1]}/_atk-tool"
    ```
*   **macOS**:
    ```zsh
    atk-tool-plus completion zsh > $(brew --prefix)/share/zsh/site-functions/_atk-tool
    ```

#### Fish

To load completions in your current shell session:
```fish
atk-tool-plus completion fish | source
```

To load completions for every new session, execute once:
```fish
atk-tool-plus completion fish > ~/.config/fish/completions/atk-tool.fish
```

#### PowerShell

To load completions in your current shell session:
```powershell
atk-tool-plus completion powershell | Out-String | Invoke-Expression
```

To load completions for every new session, add the output of the command above to your PowerShell profile.

---

## Library Usage

You can import `github.com/papelblume/atk-tool-plus` as a dependency in your own Go projects to scan and query ATK peripherals.

### Installation

```bash
go get github.com/papelblume/atk-tool-plus
```

### Code Example

```go
package main

import (
	"fmt"
	"log"

	"github.com/papelblume/atk-tool-plus"
)

func main() {
	// 1. Initialize the underlying HID library
	if err := atk.Init(); err != nil {
		log.Fatalf("Failed to init library: %v", err)
	}
	defer atk.Exit()

	// 2. Scan for supported ATK devices
	devices, err := atk.Enumerate()
	if err != nil {
		log.Fatalf("Failed to scan devices: %v", err)
	}

	if len(devices) == 0 {
		fmt.Println("No supported ATK peripherals found.")
		return
	}

	// 3. Select the first discovered peripheral
	target := devices[0]
	fmt.Printf("Connecting to %s (%s) at %s...\n", target.ModelName, target.ProductName, target.Path)

	// 4. Open a connection
	dev, err := atk.Open(target)
	if err != nil {
		log.Fatalf("Failed to open connection: %v", err)
	}
	defer dev.Close()

	// 5. Query telemetry data
	status, err := dev.QueryBattery()
	if err != nil {
		log.Fatalf("Failed to query battery status: %v", err)
	}

	fmt.Printf("Telemetry:\n")
	fmt.Printf("  Battery level: %d%%\n", status.Percentage)
	fmt.Printf("  Voltage:       %.3f V\n", status.Voltage)
	fmt.Printf("  Charging:      %t\n", status.Charging)
}
```

---

## Extending Supported Devices

Adding support for new models is straightforward. You can either register them at runtime or add them natively to the source registry.

Before adding a device, identify its UsagePage using `usbhid-dump` or a similar HID inspection tool, then match it to the correct protocol family:

| UsagePage | Protocol |
|---|---|
| `0xFF02`, `0xFF04` | v1.0 |
| `0xFF05` | v2.0 |

### 1. Adding Native Support (For Forks)

If you have forked the repository and wish to add support natively for all CLI and library users, you can add your device definition directly to the `defaultRegistry` slice in [registry.go](file:///home/mechsoull/Projects/atk-tool/registry.go):

```go
var defaultRegistry = []DeviceDefinition{
	// ... existing devices ...
	{
		Name:       "ATK F1 Extreme",
		VendorID:   0x2bdf,                      // Vendor ID
		ProductID:  0x0a0e,                      // Product ID
		UsagePages: []uint16{0xFF02, 0xFF04},    // Raw communication usage pages
		ReportID:   DefaultReportID,             // Command report ID (typically DefaultReportID)
	},
}
```

> [!NOTE]
> If the new device uses a Vendor ID different from `373b`, you will also need to add a corresponding udev rule in your Linux setup to grant user permissions for it. See the [Linux Setup](#linux-setup-udev-rules) section for the rule format.

### 2. Registering Devices at Runtime

If you are using `atk-tool-plus` as a library in your own project, you can register custom models at runtime before calling the scanning/enumeration functions:

```go
import "github.com/papelblume/atk-tool-plus"

func init() {
	atk.RegisterDevice(atk.DeviceDefinition{
		Name:       "ATK F1 Extreme",
		VendorID:   0x2bdf,
		ProductID:  0x0a0e,
		UsagePages: []uint16{0xFF02, 0xFF04},
		ReportID:   0x08,
	})
}
```

---

## Contributing

Thank you for your interest in this project! **Contributions are not accepted at this moment.**

This means pull requests, feature requests, and additions to the device registry will not be merged, regardless of scope. Bugs and unexpected behavior reports are still welcome via the issue tracker.

You are very welcome to **fork** the repository and extend it for your own devices — the [MIT License](LICENSE) explicitly permits use, modification, and redistribution. For devices not covered by the default registry, see the [Extending Supported Devices](#extending-supported-devices) section for runtime registration, or fork and edit the registry directly.

---

## Disclaimer

This repository has been developed with the assistance of Generative AI tools. While effort is made to maintain code quality, correctness, and security, users are encouraged to test and verify code behavior before deploying in production environments.
