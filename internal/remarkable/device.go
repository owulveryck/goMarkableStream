package remarkable

type DeviceModel int

const (
	UnknownDevice DeviceModel = iota
	Remarkable2
	RemarkablePaperPro
)

func (d DeviceModel) String() string {
	switch d {
	case Remarkable2:
		return "Remarkable2"
	case RemarkablePaperPro:
		return "RemarkablePaperPro"
	default:
		return "UnknownDevice"
	}
}
