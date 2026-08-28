package atk

const (
	// DefaultReportID is the query Report ID used by ATK devices.
	DefaultReportID = 0x08

	// FrameSize is the size of the V1 payload frame (excluding the Report ID).
	FrameSize = 16

	// CmdQueryBattery is the subcommand opcode for querying battery information.
	CmdQueryBattery = 0x04

	// FrameSizeV2 is the size of the V2 payload frame (excluding the Report ID).
	FrameSizeV2 = 64

	// CmdQueryBatteryV2ByteOne is the first subcommand byte for querying battery information.
	CmdQueryBatteryV2ByteOne = 0x07

	// CmdQueryBatteryV2ByteTwo is the second subcommand byte for querying battery information.
	CmdQueryBatteryV2ByteTwo = 0x01

	// CmdMarkerByte is sent with all commands at index 3 and received at index 2
	CmdMarkerByte = 0x72
)

// FinalizePayload constructs a 16-byte payload starting with the opcode,
// followed by the body, and ending with a checksum at the last byte.
// The checksum is computed such that the sum of all 16 bytes equals 0x4D.
func FinalizePayload(op byte, body [14]byte) []byte {
	payload := make([]byte, FrameSize)
	payload[0] = op
	copy(payload[1:15], body[:])

	var sum byte
	for i := 0; i < 15; i++ {
		sum += payload[i]
	}
	payload[15] = 0x4D - sum
	return payload
}
