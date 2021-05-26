package client

import (
	"bytes"
	"image"
	"image/jpeg"

	"github.com/mattn/go-mjpeg"
)

type MJPEGDisplayer struct {
	conf        *Configuration
	mjpegStream *mjpeg.Stream
}

func NewMJPEGDisplayer(c *Configuration, stream *mjpeg.Stream) *MJPEGDisplayer {
	return &MJPEGDisplayer{
		conf:        c,
		mjpegStream: stream,
	}
}

func (m *MJPEGDisplayer) Display(img *image.Gray) error {
	b := bufPool.Get().(*bytes.Buffer)
	b.Reset()
	defer bufPool.Put(b)

	var err error
	if m.conf.Colorize {
		colored := colorize(img)
		err = jpeg.Encode(b, colored, nil)
		defer releaseRGBA(colored)
	} else {
		err = jpeg.Encode(b, img, nil)
	}
	if err != nil {
		return err
	}
	err = m.mjpegStream.Update(b.Bytes())
	if err != nil {
		return err
	}

	return nil
}
