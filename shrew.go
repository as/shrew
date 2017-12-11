package shrew

import (
	"image"
	"image/draw"

	"github.com/as/frame/font"
)

type Screen interface {
	AllocImage(r image.Rectangle) Bitmap
	Mouse() chan Mouse
	Kbd() chan Kbd
	Bitmap
}
type Bitmap interface {
	draw.Image
	Draw(dst draw.Image, r image.Rectangle, src image.Image, sp image.Point, op draw.Op)
	StringBG(dst draw.Image, p image.Point, src image.Image, sp image.Point, ft *font.Font, s []byte, bg image.Image, bgp image.Point) int
	Flush(r image.Rectangle) error
}
type Mouse struct {
	Button int
	image.Point
}
type Kbd struct {
	Rune  rune
	Press int
}

type Client struct {
	W  Bitmap
	M  <-chan Mouse
	K  <-chan Kbd
	C  chan Msg
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
	Sp image.Point
	W  Bitmap
	M  chan Mouse
	K  chan Kbd
	C  chan Msg
	CO chan string
	CI chan string
}
