package shrew

import (
	"image"
	"image/draw"
)

type Screen interface {
	Bitmap
}
type Bitmap interface {
	draw.Image
	SubImage(image.Rectangle) image.Image
}
type Mouse struct {
	Button int
	image.Point
}
type Kbd int

type Client struct {
	W  Screen
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
