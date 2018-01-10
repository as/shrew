package win

import (
	"image"

	"github.com/as/shrew"
)

type cacher struct {
	buffer bool
	r      []image.Rectangle
	shrew.Bitmap
}

func (c *cacher) Flush(r ...image.Rectangle) error {
	if c.buffer {
		c.r = append(c.r, r...)
	} else {
		c.Bitmap.Flush(append(c.r, r...)...)
		c.r = c.r[:0]
	}
	return nil
}
