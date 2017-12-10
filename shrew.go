package shrew

import (
	"image"
	"image/draw"
)

type Screen interface {
	AllocImage(r image.Rectangle) Bitmap
	Mouse() chan Mouse
	Kbd() chan Kbd
	Bounds() image.Rectangle
	//Bitmap
}
type Bitmap interface {
	Bounds() image.Rectangle
	Draw(r image.Rectangle, src image.Image, sp image.Point, op draw.Op)
	Flush(r image.Rectangle) error
}
type Mouse struct {
	Button int
	image.Point
}
type Kbd int

type Client struct {
	W  Bitmap
	M  <-chan Mouse
	K  <-chan Kbd
	CO chan<- string
	CI <-chan string
}

func (c *Client) prog(e *Env) {
	defer c.Close()
	for {
	}
}
func (c *Client) Close() {
	c.CO <- "Delete"
}

type Env struct {
	W  Bitmap
	M  chan Mouse
	K  chan Kbd
	CO chan string
	CI chan string
}
