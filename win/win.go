package win

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/as/frame"
	"github.com/as/frame/font"
	"github.com/as/shrew"
	"github.com/as/text"
)

type Win struct {
	c *shrew.Client
	*frame.Frame
	text.Editor
	ScrollBar
	org      int64
	Sq       int64
	inverted int
	cacher   *cacher
	buffer   bool
}

type Options struct {
	Pad    image.Point
	Font   *font.Font
	Color  frame.Color
	Editor text.Editor
}

var defaultOptions = Options{
	Pad:   image.Pt(15, 15),
	Font:  font.NewGoMono(11),
	Color: frame.Mono,
}

func New(c *shrew.Client, opt *Options) *Win {
	if opt == nil {
		opt = &defaultOptions
	}
	ed := opt.Editor
	if ed == nil {
		ed, _ = text.Open(text.NewBuffer())
	}
	r := c.W.Bounds()
	r.Min.X += opt.Pad.X
	r.Min.Y += opt.Pad.Y
	cacher := &cacher{Bitmap: c.W}
	w := &Win{
		c:      c,
		Editor: ed,
		cacher: cacher,
		Frame:  frame.NewDrawer(r, opt.Font, c.W, opt.Color, cacher),
	}

	w.init()
	w.scrollinit(r)
	return w
}

func (w *Win) init() {
	w.Blank()
	w.Fill()
	q0, q1 := w.Dot()
	w.Select(q0, q1)
}
func (w *Win) Blank() {
	r := w.c.W.Bounds()
	w.c.W.Draw(w.c.W, r, w.Color.Back, image.ZP, draw.Src)
	w.drawsb()
}
func (w *Win) Dot() (int64, int64) {
	return w.Editor.Dot()
}
func (w *Win) BackNL(p int64, n int) int64 {
	if n == 0 && p > 0 && w.Bytes()[p-1] != '\n' {
		n = 1
	}
	for i := n; i > 0 && p > 0; {
		i--
		p--
		if p == 0 {
			break
		}
		for j := 512; j-1 > 0 && p > 0; p-- {
			j--
			if p-1 < 0 || p-1 > w.Len() || w.Bytes()[p-1] == '\n' {
				break
			}
		}
	}
	return p
}
func (w *Win) Len() int64 {
	return w.Editor.Len()
}
func (w *Win) Refresh() {
	w.Frame.Refresh()
}
func (w *Win) Bytes() []byte {
	return w.Editor.Bytes()
}

func (w *Win) Clicksb(pt image.Point, dir int) {
	w.clicksb(pt, dir)
	w.drawsb()

}
func region5(r0, r1, q0, q1 int) int {
	{
		r0 := int64(r0)
		r1 := int64(r1)
		q0 := int64(q0)
		q1 := int64(q1)
		return text.Region5(r0, r1, q0, q1)
	}
}

func clamp32(v, l, h int) int {
	if v < l {
		return l
	}
	if v > h {
		return h
	}
	return v
}

func clamp(v, l, h int64) int64 {
	if v < l {
		return l
	}
	if v > h {
		return h
	}
	return v
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
func drawBorder(dst draw.Image, r image.Rectangle, src image.Image, sp image.Point, thick int) {
	draw.Draw(dst, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+thick), src, sp, draw.Src)
	draw.Draw(dst, image.Rect(r.Min.X, r.Max.Y-thick, r.Max.X, r.Max.Y), src, sp, draw.Src)
	draw.Draw(dst, image.Rect(r.Min.X, r.Min.Y, r.Min.X+thick, r.Max.Y), src, sp, draw.Src)
	draw.Draw(dst, image.Rect(r.Max.X-thick, r.Min.Y, r.Max.X, r.Max.Y), src, sp, draw.Src)
}

// Put

var (
	Red    = image.NewUniform(color.RGBA{255, 0, 0, 255})
	Green  = image.NewUniform(color.RGBA{255, 255, 192, 25})
	Blue   = image.NewUniform(color.RGBA{0, 192, 192, 255})
	Cyan   = image.NewUniform(color.RGBA{234, 255, 255, 255})
	White  = image.NewUniform(color.RGBA{255, 255, 255, 255})
	Yellow = image.NewUniform(color.RGBA{255, 255, 224, 255})
	X      = image.NewUniform(color.RGBA{255 - 32, 255 - 32, 224 - 32, 255})
	LtGray = image.NewUniform(color.RGBA{66*2 + 25, 66*2 + 25, 66*2 + 35, 255})
	Gray   = image.NewUniform(color.RGBA{66, 66, 66, 255})
	Mauve  = image.NewUniform(color.RGBA{0x99, 0x99, 0xDD, 255})
)
