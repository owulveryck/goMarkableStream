package streammock

import (
	"sync"
	"time"
)

// Server implementation
type Server struct {
	imagePool sync.Pool
	runnable  chan struct{}
}

// Start the pooling thread
func (s *Server) Start() {
	go func(c chan struct{}) {
		for {
			c <- struct{}{}
			time.Sleep(200 * time.Millisecond)
		}
	}(s.runnable)
}

// NewServer ...
func NewServer() *Server {
	return &Server{
		imagePool: sync.Pool{
			New: func() interface{} {
				return &Image{
					Width:     ScreenWidth,
					Height:    ScreenHeight,
					ImageData: make([]byte, ScreenWidth*ScreenHeight),
				}
			},
		},
		runnable: make(chan struct{}),
	}
}

// GetImage input is nil
func (s *Server) GetImage(_ *Input, stream Stream_GetImageServer) error {
	for range s.runnable {
		img := s.imagePool.Get().(*Image)
		for i := 0; i < len(img.ImageData); i++ {
			img.ImageData[i] = byte(time.Now().Second())
		}
		if err := stream.Send(img); err != nil {
			return err
		}
	}
	return nil
}
