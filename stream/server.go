package stream

import (
	context "context"
	"encoding/binary"
	io "io"
	"sync"
)

// Server implementation
type Server struct {
	imagePool   sync.Pool
	r           io.ReaderAt
	pointerAddr int64
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
	}
}

// GetImage input is nil
func (s *Server) GetImage(ctx context.Context, in *Input) (*Image, error) {
	img := s.imagePool.Get().(*Image)
	_, err := s.r.ReadAt(img.ImageData, s.pointerAddr)
	if err != nil {
		s.imagePool.Put(img)
		return nil, err
	}
	return img, nil
}

func getPointer(r io.ReaderAt, offset int64) (int64, error) {
	pointer := make([]byte, 4)
	_, err := r.ReadAt(pointer, offset)
	if err != nil {
		return 0, err
	}
	return int64(binary.LittleEndian.Uint32(pointer)), nil
}
