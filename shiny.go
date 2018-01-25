package shrew

import (
	"image"
	"image/color"
	"image/draw"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/as/frame"
	"github.com/as/memdraw"
	"github.com/as/shiny/screen"
	"github.com/as/ui"
	"github.com/golang/freetype/truetype"
	//	"golang.org/x/exp/shiny/screen"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
)

var (
	noCrappyOptimizations = false
)

type ShinyScreen struct {
	dev *ui.Dev
	K   chan Kbd
	M   chan Mouse
	ft  *truetype.Font
	Bitmap
}

func (s *ShinyScreen) AllocImage(r image.Rectangle) Bitmap {
	b := s.dev.NewBuffer(r.Size())
	bmp := &ShinyBitmap{
		sp:        r.Min,
		size:      r.Size(),
		b:         b,
		w:         s.dev.Window(),
		ctl:       make(chan Msg),
		ctl2:      make(chan Msg),
		stringBGC: make(chan txStringBG),
		replyint:  make(chan int),
	}
	go bmp.run()
	return bmp
}
func (s *ShinyScreen) Bounds() image.Rectangle { return image.Rect(0, 0, 2500, 1400) }
func (s *ShinyScreen) Kbd() chan Kbd           { return s.K }
func (s *ShinyScreen) Mouse() chan Mouse       { return s.M }

/*
CodeLeftControl  Code = 224
CodeLeftShift    Code = 225
CodeLeftAlt      Code = 226
CodeLeftGUI      Code = 227
CodeRightControl Code = 228
CodeRightShift   Code = 229
CodeRightAlt     Code = 230
CodeRightGUI     Code = 231
*/

func ShinyClient() *ShinyScreen {
	dev, err := ui.Init(nil)
	if err != nil {
		panic(err)
	}
	w := dev.Window()
	K := make(chan Kbd, 1)
	M := make(chan Mouse, 1)
	sc := &ShinyScreen{
		dev: dev,
		K:   K,
		M:   M,
	}
	sc.fontinit()
	sc.Bitmap = sc.AllocImage(sc.Bounds())
	mstate := Mouse{}
	go func() {
		for {
			switch e := w.NextEvent().(type) {
			case mouse.Event:

				// Some silly operating system though it would
				// be clever to send an event for a step of a mouse
				// wheel. We fix that here.
				if e.Button < 0 {
					e.Button = -e.Button + 3
					mstate.Button |= 1 << uint(e.Button-1)
					M <- mstate
					mstate.Button &^= 1 << uint(e.Button-1)
					continue
				}
				if e.Direction == 1 {
					mstate.Button |= 1 << uint(e.Button-1)
				} else if e.Direction == 2 {
					mstate.Button &^= 1 << uint(e.Button-1)
				}
				mstate.X = int(e.X)
				mstate.Y = int(e.Y)
				M <- mstate
			case key.Event:
				if e.Code == key.CodeRightShift || e.Code == key.CodeLeftShift {
					continue
				}
				press := 0
				if e.Direction == 1 || e.Direction == 0 {
					press = 1

				}
				if e.Rune == 13 {
					e.Rune = 10
				}
				K <- Kbd{
					Rune:  e.Rune,
					Press: press,
				}
			case paint.Event:
			case interface{}:
			}
		}
	}()
	return sc
}

type ShinyBitmap struct {
	sp        image.Point
	size      image.Point
	b         screen.Buffer
	w         screen.Window
	ctl       chan Msg // for drawing
	ctl2      chan Msg // for wsys
	stringBGC chan txStringBG
	wg        sync.WaitGroup
	refresh   chan Msg
	draw      chan Msg
	replyint  chan int
	//ff      font.Face
}

func (s *ShinyBitmap) Set(x, y int, v color.Color) {
	s.b.RGBA().Set(x, y, v)
}
func (s *ShinyBitmap) At(x, y int) color.Color {
	return s.b.RGBA().At(x, y)
}
func (s *ShinyBitmap) ColorModel() color.Model {
	return s.b.RGBA().ColorModel()
}

type txStringBG struct {
	dst  draw.Image
	p    image.Point
	src  image.Image
	sp   image.Point
	ft   font.Face
	data []byte
	bg   image.Image
	bgp  image.Point
}

type Msg struct {
	string
	Kind string
	Sp   image.Point
	Name string
	kind byte
	dst  draw.Image
	p    image.Point
	r    image.Rectangle
	rs   []image.Rectangle
	src  image.Image
	sp   image.Point
	pt0  image.Point
	pt1  image.Point
	pts  []image.Point
	end0 int
	end1 int
	op   draw.Op
	//	replyc   chan error
	replyint chan int
	data     []byte
	ft       font.Face
	bg       image.Image
	bgp      image.Point
	thick    int
}

func (m *Msg) Canon(s *ShinyBitmap) {
	if m.dst == s || m.dst == nil {
		m.dst = s.b.RGBA()
	}
	if m.src == s || m.src == nil {
		m.src = s.b.RGBA()
	}
	if m.bg == s {
		m.bg = s.b.RGBA()
	}
}

func (s *ShinyBitmap) Move(sp image.Point) {
	s.ctl2 <- Msg{
		string: "move",
		sp:     sp,
	}
}

var BG = image.NewUniform(color.RGBA{77, 77, 77, 255})

func (s *ShinyBitmap) BG() {
	draw.Draw(s.b.RGBA(), s.b.Bounds(), BG, image.ZP, draw.Src)
	s.w.Upload(s.sp, s.b, s.b.Bounds())
}

func (s *ShinyBitmap) run() {
	s.BG()
	for {
		select {
		case Msg := <-s.stringBGC:
			s.replyint <- s.stringBG2(&Msg)
		case Msg := <-s.ctl2:
			if Msg.string == "move" {
				s.sp = Msg.sp
			}
		case Msg := <-s.ctl:
			(&Msg).Canon(s)
			switch Msg.kind {
			case 'd':
				draw.Draw(Msg.dst, Msg.r, Msg.src, Msg.sp, Msg.op)
			case 'x':
				Msg.replyint <- s.stringBG(Msg.dst, Msg.p, Msg.src, Msg.sp, Msg.ft, Msg.data, Msg.bg, Msg.bgp)
			case '1':
				s.bezier(Msg.dst, Msg.pts, Msg.end0, Msg.end1, Msg.thick, Msg.src, Msg.sp)
			case '2':
				s.bspline(Msg.dst, Msg.pts, Msg.end0, Msg.end1, Msg.thick, Msg.src, Msg.sp)
			case 'P':
				s.poly(Msg.dst, Msg.pts, Msg.end0, Msg.end1, Msg.thick, Msg.src, Msg.sp)
			case 'L':
				s.line(Msg.dst, Msg.pt0, Msg.pt1, Msg.thick, Msg.src, Msg.sp)
			case 'f':
				if noCrappyOptimizations {
					s.w.Upload(s.sp, s.b, s.b.Bounds())
					continue
				}
				var wg sync.WaitGroup
				wg.Add(len(Msg.rs))
				for _, r := range Msg.rs {
					r := r
					dp := s.sp.Add(r.Min)
					go func() {
						s.w.Upload(dp, s.b, r)
						wg.Done()
					}()
				}
				wg.Wait()
			}
		}
	}
}

func (s *ShinyBitmap) Line(dst draw.Image, pt0, pt1 image.Point, thick int, src image.Image, sp image.Point) {
	s.ctl <- Msg{
		kind:  'L',
		dst:   dst,
		pt0:   pt0,
		pt1:   pt1,
		thick: thick,
		src:   src,
		sp:    sp,
	}
}

func (s *ShinyBitmap) Bspline(dst draw.Image, pts []image.Point, end0, end1, thick int, src image.Image, sp image.Point) {
	s.ctl <- Msg{
		kind:  '2',
		dst:   dst,
		pts:   pts,
		end0:  end0,
		end1:  end1,
		thick: thick,
		src:   src,
		sp:    sp,
	}
}

func (s *ShinyBitmap) Bezier(dst draw.Image, pts []image.Point, end0, end1, thick int, src image.Image, sp image.Point) {
	s.ctl <- Msg{
		kind:  '1',
		dst:   dst,
		pts:   pts,
		end0:  end0,
		end1:  end1,
		thick: thick,
		src:   src,
		sp:    sp,
	}
}

func (s *ShinyBitmap) Poly(dst draw.Image, pts []image.Point, end0, end1, thick int, src image.Image, sp image.Point) {
	s.ctl <- Msg{
		kind:  'P',
		dst:   dst,
		pts:   pts,
		end0:  end0,
		end1:  end1,
		thick: thick,
		src:   src,
		sp:    sp,
	}
}

func (s *ShinyBitmap) bezier(dst draw.Image, p []image.Point, end0, end1, thick int, src image.Image, sp image.Point) {
	memdraw.Bezier(dst, p[0], p[1], p[2], p[3], end0, end1, thick, src, sp)
}
func (s *ShinyBitmap) bspline(dst draw.Image, p []image.Point, end0, end1, thick int, src image.Image, sp image.Point) {
	memdraw.Bezspline(dst, p[0], p[1], p[2], p[3], end0, end1, thick, src, sp)
}

func (s *ShinyBitmap) poly(dst draw.Image, p []image.Point, end0, end1, thick int, src image.Image, sp image.Point) {
	memdraw.Poly(dst, p, end0, end1, thick, src, sp)
}

func (s *ShinyBitmap) line(dst draw.Image, pt0, pt1 image.Point, thick int, src image.Image, sp image.Point) {
	memdraw.Line(dst, pt0, pt1, thick, src, sp)
}

func (s *ShinyBitmap) Bounds() image.Rectangle {
	return s.b.Bounds() //.Add(s.sp)
}
func (s *ShinyBitmap) Draw(dst draw.Image, r image.Rectangle, src image.Image, sp image.Point, op draw.Op) {
	s.ctl <- Msg{
		kind: 'd',
		dst:  dst,
		r:    r,
		src:  src,
		sp:   sp,
		op:   op,
	}
}
func (s *ShinyBitmap) DrawBytes(dst draw.Image, dot image.Point, src image.Image, data []byte) {
	s.ctl <- Msg{
		kind: 'x',
		dst:  dst,
		src:  src,
		sp:   dot,
		data: data,
	}
}

func (z *ShinyBitmap) stringBG2(o *txStringBG) int {
	dst, p, ft, s, src, sp, bg, bgp := o.dst, o.p, o.ft, o.data, o.src, o.sp, o.bg, o.bgp

	if ft, ok := ft.(StaticFace); ok && bg != nil {
		return staticStringBG(z.b.RGBA(), p, ft, s, src.(*image.Uniform).C, bg.(*image.Uniform).C)
	}
	p.Y += frame.Ascent(ft)
	for _, b := range s {
		dr, mask, maskp, advance, ok := ft.Glyph(fixed.P(p.X, p.Y), rune(b))
		if !ok {
			//panic("RuneBG")
		}
		if bg != nil {
			draw.Draw(dst, dr, bg, bgp, draw.Src)
		}
		draw.DrawMask(dst, dr, src, sp, mask, maskp, draw.Over)
		p.X += frame.Fix(advance)
	}
	return p.X
}

func (*ShinyBitmap) stringBG(dst draw.Image, p image.Point, src image.Image, sp image.Point, ft font.Face, s []byte, bg image.Image, bgp image.Point) int {
	if ft, ok := ft.(StaticFace); ok {
		if bg, ok := bg.(*image.Uniform); ok {
			if bg, ok := bg.C.(color.RGBA); ok {
				return staticStringBG(dst, p, ft, s, src.(*image.Uniform).C.(color.RGBA), bg)
			}
		}
	}
	p.Y += frame.Ascent(ft)
	for _, b := range s {
		dr, mask, maskp, advance, ok := ft.Glyph(fixed.P(p.X, p.Y), rune(b))
		if !ok {
			//panic("RuneBG")
		}
		//draw.Draw(dst, dr, bg, bgp, draw.Src)
		draw.DrawMask(dst, dr, src, sp, mask, maskp, draw.Over)
		p.X += frame.Fix(advance)
		if len(s)-1 == 0 {
			break
		}
		s = s[1:]
	}
	return p.X
}

func staticStringBG(dst draw.Image, p image.Point, ft StaticFace, s []byte, fg, bg color.Color) int {
	//p.Y += frame.Ascent(ft)
	r := image.Rectangle{p, p}
	r.Max.Y += frame.Dy(ft)
	for _, b := range s {
		img := ft.RawGlyph(b, fg, bg)
		dx := img.Bounds().Dx()
		r.Max.X += dx
		draw.Draw(dst, r, img, img.Bounds().Min, draw.Src)
		r.Min.X += dx //img.Bounds().Dx() //+ ft.stride //Add(image.Pt(img.Bounds().Dx(), 0))
	}
	return r.Min.X
}

type StaticFace interface {
	font.Face
	RawGlyph(b byte, fg, bg color.Color) image.Image
}

func (s *ShinyBitmap) StringBG(dst draw.Image, p image.Point, src image.Image, sp image.Point, ft font.Face, data []byte, bg image.Image, bgp image.Point) int {
	s.stringBGC <- txStringBG{dst, p, src, sp, ft, data, bg, bgp}
	return <-s.replyint
	//	s.ctl <- Msg{
	//		kind:     'x',
	//		dst:      dst,
	//		p:        p,
	//		src:      src,
	//		sp:       p,
	//		ft:       ft,
	///		data:     data,
	//		bg:       bg,
	//		bgp:      bgp,
	//		replyint: replyint,
	//	}
	//	return <-replyint
}

func (s *ShinyBitmap) Flush(r ...image.Rectangle) error {
	if len(r) == 0 {
		return nil
	}
	s.ctl <- Msg{
		kind: 'f',
		rs:   r,
	}
	return nil
}
