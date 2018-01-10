package win

import (
	"image"
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

type Config struct {
	Pad    image.Point
	Font   *font.Font
	Color  frame.Color
	Editor text.Editor
}

var defaultOptions = Config{
	Pad:   image.Pt(15, 15),
	Font:  font.NewGoMono(11),
	Color: frame.Mono,
}

func New(c *shrew.Client, opt *Config) *Win {
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
