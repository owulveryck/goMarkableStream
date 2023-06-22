package streammock

//go:generate protoc --gofast_out=plugins=grpc:.  defs.proto3

const (
	// ScreenWidth of the remarkable 2
	ScreenWidth = 1872
	// ScreenHeight of the remarkable 2
	ScreenHeight = 1404
)
