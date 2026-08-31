//go:build !rmmove

package remarkable

func DetectDevice() (DeviceModel, string, error) {
	return Model, Model.String(), nil
}
