package main

import (
	"image"
	"image/color"
	"image/draw"
	"io"

	"github.com/as/frame"
	"github.com/as/frame/font"
	"github.com/as/shrew"
	"github.com/as/text"
)

type Win struct {
	c *shrew.client
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

type cacher struct {
	buffer bool
	r      []image.Rectangle
	shrew.Bitmap
}

func (w *Win) Buffer() {
	w.cacher.buffer = true
}
func (w *Win) Unbuffer() {
	w.cacher.buffer = false
	w.Flush()
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

func New(c *shrew.client, opt *Options) *Win {
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

func (w *Win) WriteAt(p []byte, at int64) (n int, err error) {
	n, err = w.Editor.(io.WriterAt).WriteAt(p, at)
	q0, q1 := at, at+int64(len(p))

	switch text.Region5(q0, q1, w.org-1, w.org+w.Frame.Len()+1) {
	case -2:
		// Logically adjust origin to the left (up)
		w.org -= q1 - q0
	case -1:
		// Remove the visible text and adjust left
		w.Frame.Delete(0, q1-w.org)
		w.org = q0
		w.Fill()
	case 0:
		p0 := clamp(q0-w.org, 0, w.Frame.Len())
		p1 := clamp(q1-w.org, 0, w.Frame.Len())
		w.Frame.Delete(p0, p1)
		w.Fill()
	case 1:
		w.Frame.Delete(q0-w.org, w.Frame.Len())
		w.Fill()
	case 2:
	}
	return
}

// Insert inserts the bytes in p at position q0. When q0
// is zero, Insert prepends the bytes in p to the underlying
// buffer
func (w *Win) Insert(p []byte, q0 int64) (n int) {
	if len(p) == 0 {
		return 0
	}

	// If at least one point in the region overlaps the
	// frame's visible area then we alter the frame. Otherwise
	// there's no point in moving text down, it's just annoying.

	switch q1 := q0 + int64(len(p)); text.Region5(q0, q1, w.org-1, w.org+w.Frame.Len()+1) {
	case -2:
		w.org += q1 - q0
	case -1:
		// Insertion to the left
		w.Frame.Insert(p[q1-w.org:], 0)
		w.org += w.org - q0
	case 1:
		w.Frame.Insert(p, q0-w.org)
	case 0:
		if q0 < w.org {
			p0 := w.org - q0
			w.Frame.Insert(p[p0:], 0)
			w.org += w.org - q0
		} else {
			w.Frame.Insert(p, q0-w.org)
		}
	}
	if w.Editor == nil {
		panic("nil editor")
	}
	n = w.Editor.Insert(p, q0)
	return n
}

const (
	// Extra lines to scroll down to comfortably display the result of a look operation
	JumpScrollMargin = -3
)

// Select selects the range [q0:q1] inclusive
func (w *Win) Select(q0, q1 int64) {
	if q0 > q1 {
		q0, q1 = q1, q0
	}
	q00, q11 := w.Dot()
	w.Editor.Select(q0, q1)
	reg := text.Region3(q0, w.org-1, w.org+w.Frame.Len())
	if q00 == q0 && q11 == q1 {
		//return
	}
	p0, p1 := q0-w.org, q1-w.org
	w.Frame.Select(p0, p1)
	if q0 == q1 && reg != 0 {
		//w.Untick()	// TODO(as): win.exe cursor disappeared when this was uncommented
	}
}

// Jump scrolls the active selection into view. An optional mouseFunc
// is given the transfer coordinates to move the mouse cursor under
// the selection.
func (w *Win) Jump(mouseFunc func(image.Point)) {
	q0, q1 := w.Dot()
	if text.Region5(q0, q1, w.Origin(), w.Origin()+w.Frame.Len()) != 0 {
		w.SetOrigin(q0, true)
		w.Scroll(JumpScrollMargin)
	}
	if mouseFunc != nil {
		jmp := w.PointOf(q0 - w.org)
		mouseFunc(w.Bounds().Min.Add(jmp))
	}
}

func (w *Win) Origin() int64 {
	return w.org
}

// Delete deletes the range [q0:q1] inclusive. If there
// is nothing to delete, it returns 0.
func (w *Win) Delete(q0, q1 int64) (n int) {
	if w.Len() == 0 {
		return 0
	}
	if q0 > q1 {
		q0, q1 = q1, q0
	}
	if q1 > w.Len() {
		q1 = w.Len()
	}
	w.Editor.Delete(q0, q1)

	switch text.Region5(q0, q1, w.org-1, w.org+w.Frame.Len()+1) {
	case -2:
		// Logically adjust origin to the left (up)
		w.org -= q1 - q0
	case -1:
		// Remove the visible text and adjust left
		w.Frame.Delete(0, q1-w.org)
		w.org = q0
		w.Fill()
	case 0:
		p0 := clamp(q0-w.org, 0, w.Frame.Len())
		p1 := clamp(q1-w.org, 0, w.Frame.Len())
		w.Frame.Delete(p0, p1)
		w.Fill()
	case 1:
		w.Frame.Delete(q0-w.org, w.Frame.Len())
		w.Fill()
	case 2:
	}
	return int(q1 - q0)
}

func (w *Win) fixEnd() {
	fr := w.Frame.Bounds()
	if pt := w.PointOf(w.Frame.Len()); pt.Y != fr.Max.Y {
		w.Paint(pt, fr.Max, w.Frame.Color.Palette.Back)
	}
}

func (w *Win) Fill() {
	if w.Frame.Full() {
		return
	}
	for !w.Frame.Full() {
		qep := w.org + w.Nchars
		n := max(0, min(w.Len()-qep, 2000))
		if n == 0 {
			break
		}
		rp := w.Bytes()[qep : qep+n]
		m := len(rp)
		nl := w.MaxLine() - w.Line()
		m = 0
		i := int64(0)
		for i < n {
			if rp[i] == '\n' {
				m++
				if m >= nl {
					i++
					break
				}
			}
			i++
		}
		w.Frame.Insert(rp[:i], w.Nchars)
	}
	w.Flush()
}

func (w *Win) SetOrigin(org int64, exact bool) {
	org = clamp(org, 0, w.Len())
	if org == w.org {
		return
	}
	//	w.Mark()
	if org > 0 && !exact {
		for i := 0; i < 2048 && org < w.Len(); i++ {
			if w.Bytes()[org] == '\n' {
				org++
				break
			}
			org++
		}
	}
	w.setOrigin(clamp(org, 0, w.Len()))
}

func (w *Win) setOrigin(org int64) {
	if org == w.org {
		return
	}
	fl := w.Frame.Len()
	switch text.Region5(org, org+fl, w.org, w.org+fl) {
	case -1:
		// Going down a bit
		w.Frame.Insert(w.Bytes()[org:org+(w.org-org)], 0)
		w.org = org
	case -2, 2:
		w.Frame.Delete(0, w.Frame.Len())
		w.org = org
		w.Fill()
	case 1:
		// Going up a bit
		w.Frame.Delete(0, org-w.org)
		w.org = org
		w.Fill()
		//w.fixEnd()

	case 0:
		panic("never happens")
	}
	q0, q1 := w.Dot()
	w.drawsb()
	w.Select(q0, q1)
}

const minSbWidth = 9

type ScrollBar struct {
	bar     image.Rectangle
	Scrollr image.Rectangle
	lastbar image.Rectangle
}

func (w *Win) scrollinit(r image.Rectangle) {
	s := w.c.W.Bounds()
	r.Max, r.Min = r.Min, s.Min
	r.Max.Y = s.Max.Y
	r.Max.X = r.Min.X + minSbWidth
	w.Scrollr = r
	w.updatesb()
	w.drawsb()
}

func (w *Win) Scroll(dl int) {
	if dl == 0 {
		return
	}
	org := w.org
	if dl < 0 {
		org = w.BackNL(org, -dl)
		w.SetOrigin(org, true)
	} else {
		if org+w.Frame.Nchars == w.Len() {
			return
		}
		r := w.Frame.Bounds()
		mul := int64(dl / w.Frame.Line())
		if mul == 0 {
			mul++
		}
		dx := w.IndexOf(image.Pt(r.Min.X, r.Min.Y+dl*w.Font.Dy())) * mul
		org += dx
		w.SetOrigin(org, false)
	}
	w.updatesb()
	w.drawsb()

}

func region3(r, q0, q1 int) int {
	return text.Region3(int64(r), int64(q0), int64(q1))
}
func (w *Win) Clicksb(pt image.Point, dir int) {
	w.clicksb(pt, dir)
	w.drawsb()

}
func (w *Win) clicksb(pt image.Point, dir int) {
	var (
		rat float64
	)
	fl := float64(w.Frame.Len())
	n := w.org
	barY1 := float64(w.bar.Max.Y)
	ptY := float64(pt.Y)
	switch dir {
	case -1:
		rat = barY1 / ptY
		delta := int64(fl * rat)
		n -= delta
	case 0:
		rat := float64(pt.Y) / float64(w.Scrollr.Dy())
		w.SetOrigin(int64(float64(w.Len())*rat), false)
		w.updatesb()
		return
	case 1:
		rat = (barY1 / ptY)
		delta := int64(fl * rat)
		n += delta
	}
	w.SetOrigin(n, false)
	w.updatesb()
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

func (w *Win) drawsb() {
	if w.Scrollr == image.ZR {
		return
	}
	if w.bar == w.lastbar {
		return
	}
	r0, r1, q0, q1 := w.bar.Min.Y, w.bar.Max.Y, w.lastbar.Min.Y, w.lastbar.Max.Y
	w.lastbar = w.bar
	r := w.bar

	drawfn := func(r image.Rectangle, c image.Image) {
		r.Min.X = w.Scrollr.Min.X
		r.Max.X = w.Scrollr.Max.X
		if r.Max.Y == 0 {
			r.Max.Y = w.Scrollr.Max.Y
		}
		w.c.W.Draw(w.c.W, r, c, image.ZP, draw.Src)
		w.c.W.Flush(r)
	}
	switch region5(r0, r1, q0, q1) {
	case -2, 2, 0:
		drawfn(image.Rect(r.Min.X, q0, r.Max.X, q1), frame.ATag0.Back)
		drawfn(image.Rect(r.Min.X, r0, r.Max.X, r1), LtGray)
	case -1:
		drawfn(image.Rect(r.Min.X, r1, r.Max.X, q1), frame.ATag0.Back)
		drawfn(image.Rect(r.Min.X, r0, r.Max.X, q0), LtGray)
	case 1:
		drawfn(image.Rect(r.Min.X, q0, r.Max.X, r0), frame.ATag0.Back)
		drawfn(image.Rect(r.Min.X, q1, r.Max.X, r1), LtGray)
		//	case 0:
		//		col := frame.ATag0.Back // for a shrinking bar
		//		if r0 < q0 {            // bar grows larger
		//			col = LtGray
		//		}
		//		w.c.W.Draw(w.Frame.RGBA(), image.Rect(r.Min.X, r0, r.Max.X, q0), col, image.ZP, draw.Src)
		//		w.c.W.Draw(w.Frame.RGBA(), image.Rect(r.Min.X, q1, r.Max.X, r1), col, image.ZP, draw.Src)
	}
}
func (w *Win) refreshsb() {
	w.c.W.Draw(w.c.W, w.Scrollr, frame.ATag0.Back, image.ZP, draw.Src)
	w.c.W.Draw(w.c.W, w.bar, LtGray, image.ZP, draw.Src)
	w.c.W.Flush(w.Scrollr, w.bar)
}

func (w *Win) updatesb() {
	r := w.Scrollr
	if r == image.ZR {
		return
	}
	rat0 := float64(w.org) / float64(w.Len()) // % scrolled
	r.Min.Y += int(float64(r.Max.Y) * rat0)

	rat1 := float64(w.org+w.Frame.Len()) / float64(w.Len()) // % covered by screen
	r.Max.Y = int(float64(r.Max.Y) * rat1)                  //int(dy * rat1)
	if have := r.Max.Y - r.Min.Y; have < 3 {
		r.Max.Y = r.Min.Y + 3
	}

	r.Min.Y = clamp32(r.Min.Y, w.Scrollr.Min.Y, w.Scrollr.Max.Y)
	r.Max.Y = clamp32(r.Max.Y, w.Scrollr.Min.Y, w.Scrollr.Max.Y)
	w.lastbar = w.bar
	w.bar = r
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
