package client

import (
	"bytes"
	"image"
	"image/jpeg"

	"github.com/mattn/go-mjpeg"
)

// MJPEGDisplayer implements the Displayer interface
type MJPEGDisplayer struct {
	conf        *Configuration
	mjpegStream *mjpeg.Stream
}

// NewMJPEGDisplayer from a configuration adds images to stream
func NewMJPEGDisplayer(c *Configuration, stream *mjpeg.Stream) *MJPEGDisplayer {
	return &MJPEGDisplayer{
		conf:        c,
		mjpegStream: stream,
	}
}

// Display adds the image to the stream
func (m *MJPEGDisplayer) Display(img *image.Gray) error {
	b := bufPool.Get().(*bytes.Buffer)
	b.Reset()
	defer bufPool.Put(b)

	var err error
	switch {
	case m.conf.Colorize:
		colored := colorize(img)
		err = jpeg.Encode(b, colored, nil)
		defer releaseRGBA(colored)
	case m.conf.Highlight:
		colored := highlight(img)
		err = jpeg.Encode(b, colored, nil)
		defer releaseRGBA(colored)
	default:
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
