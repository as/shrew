package shrew

import (
	"image"
	"image/draw"

	"golang.org/x/image/font"
)

type Screen interface {
	AllocImage(r image.Rectangle) Bitmap
	Mouse() chan Mouse
	Kbd() chan Kbd
	Bitmap
}
type Client interface {
	Name() string
	Bitmap
	M() <-chan Mouse
	K() <-chan Kbd
	C() chan Msg
}
type Wsys interface {
	//	Client
	NewClient(*Options) Client
}

type Bitmap interface {
	draw.Image
	Draw(dst draw.Image, r image.Rectangle, src image.Image, sp image.Point, op draw.Op)
	Line(dst draw.Image, pt0, pt1 image.Point, thick int, src image.Image, sp image.Point)
	StringBG(dst draw.Image, p image.Point, src image.Image, sp image.Point, ft font.Face, s []byte, bg image.Image, bgp image.Point) int
	Flush(r ...image.Rectangle) error
	Bezier(dst draw.Image, pts []image.Point, end0, end1, thick int, src image.Image, sp image.Point)
	Poly(dst draw.Image, pts []image.Point, end0, end1, thick int, src image.Image, sp image.Point)
	Bspline(dst draw.Image, pts []image.Point, end0, end1, thick int, src image.Image, sp image.Point)
}
type Mouse struct {
	Button int
	image.Point
}
type Kbd struct {
	Rune  rune
	Press int
}

func (c *client) Bounds() image.Rectangle {
	return c.Bitmap.Bounds()
}

type client struct {
	name string
	Bitmap
	m  <-chan Mouse
	k  <-chan Kbd
	c  chan Msg
	CO chan<- string
	CI <-chan string
}

func (c *client) K() <-chan Kbd   { return c.k }
func (c *client) M() <-chan Mouse { return c.m }
func (c *client) C() chan Msg     { return c.c }
func (c *client) Name() string    { return c.name }

func (c *client) prog(e *Env) {
	defer c.Close()
	for {
	}
}
func (c *client) Close() {
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
