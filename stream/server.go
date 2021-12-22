package stream

import (
	io "io"
	"sync"
	"time"
)

// Server implementation
type Server struct {
	imagePool   sync.Pool
	r           io.ReaderAt
	pointerAddr int64
	runnable    chan struct{}
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
func NewServer(r io.ReaderAt, addr int64) *Server {
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
		r:           r,
		pointerAddr: addr,
		runnable:    make(chan struct{}),
	}
}

// GetImage input is nil
func (s *Server) GetImage(_ *Input, stream Stream_GetImageServer) error {
	for range s.runnable {
		img := s.imagePool.Get().(*Image)
		_, err := s.r.ReadAt(img.ImageData, s.pointerAddr)
		if err != nil {
			s.imagePool.Put(img)
			return err
		}
		if err := stream.Send(img); err != nil {
			return err
		}
	}
	return nil
}
