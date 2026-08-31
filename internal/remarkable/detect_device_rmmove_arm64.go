//go:build linux && arm64 && rmmove

package remarkable

func DetectDevice() (DeviceModel, string, error) {
	name, err := detectChiappa("/sys/devices/soc0/machine", "/proc/device-tree/model")
	if err != nil {
		return UnknownDevice, name, err
	}
	return RemarkablePaperProMove, name, nil
}
