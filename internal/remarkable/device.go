package remarkable

// DeviceModel represents the type of reMarkable device being used
type DeviceModel int

const (
	// UnknownDevice represents an unidentified reMarkable device
	UnknownDevice DeviceModel = iota
	// Remarkable1 represents the reMarkable 1 device
	Remarkable1
	// Remarkable2 represents the reMarkable 2 device
	Remarkable2
	// RemarkablePaperPro represents the reMarkable Paper Pro device
	RemarkablePaperPro
)

func (d DeviceModel) String() string {
	switch d {
	case Remarkable1:
		return "Remarkable1"
	case Remarkable2:
		return "Remarkable2"
	case RemarkablePaperPro:
		return "RemarkablePaperPro"
	default:
		return "UnknownDevice"
	}
}
