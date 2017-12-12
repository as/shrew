package shrew

import (
	"image"
	"image/color"
	"image/draw"
	"sync"

	"github.com/as/frame/font"
	"github.com/as/memdraw"
	"github.com/as/ui"
	"github.com/golang/freetype/truetype"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
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
		sp:   r.Min,
		size: r.Size(),
		b:    b,
		w:    s.dev.Window(),
		ctl:  make(chan Msg, 11),
		ctl2: make(chan Msg),
	}
	go bmp.run()
	return bmp
}
func (s *ShinyScreen) Bounds() image.Rectangle { return image.Rect(0, 0, 2500, 1400) }
func (s *ShinyScreen) Kbd() chan Kbd           { return s.K }
func (s *ShinyScreen) Mouse() chan Mouse       { return s.M }

func ShinyClient() *ShinyScreen {
	dev, err := ui.Init(nil)
	if err != nil {
		panic(err)
	}
	w := dev.Window()
	K := make(chan Kbd, 25)
	M := make(chan Mouse, 25)
	sc := &ShinyScreen{
		dev: dev,
		K:   K,
		M:   M,
	}
	sc.fontinit()
	//sc.Bitmap = sc.AllocImage(image.Rect(0, 0, 1024, 768))
	mstate := Mouse{}
	go func() {
		for {
			switch e := w.NextEvent().(type) {
			case mouse.Event:
				if e.Direction == 1 {
					mstate.Button |= 1 << uint(e.Button-1)
				} else if e.Direction == 2 {
					mstate.Button &^= 1 << uint(e.Button-1)
				}
				mstate.X = int(e.X)
				mstate.Y = int(e.Y)
				M <- mstate
			case key.Event:
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
	sp      image.Point
	size    image.Point
	b       screen.Buffer
	w       screen.Window
	ctl     chan Msg // for drawing
	ctl2    chan Msg // for wsys
	wg      sync.WaitGroup
	refresh chan Msg
	draw    chan Msg
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

type Msg struct {
	string
	Kind     string
	Sp       image.Point
	Name     string
	kind     byte
	dst      draw.Image
	p        image.Point
	r        image.Rectangle
	src      image.Image
	sp       image.Point
	pt0      image.Point
	pt1      image.Point
	pts      []image.Point
	end0     int
	end1     int
	op       draw.Op
	replyc   chan error
	replyint chan int
	data     []byte
	ft       *font.Font
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

func (s *ShinyBitmap) run() {
	draw.Draw(s.b.RGBA(), s.b.Bounds(), image.NewUniform(color.RGBA{77, 77, 77, 255}), image.ZP, draw.Src)
	s.w.Upload(s.sp, s.b, s.b.Bounds())
	for {
		select {
		case Msg := <-s.ctl2:
			if Msg.string == "move" {
				s.sp = Msg.sp
			}
		case Msg := <-s.ctl:
			(&Msg).Canon(s)
			switch Msg.kind {
			case '1':
				s.bezier(Msg.dst, Msg.pts, Msg.end0, Msg.end1, Msg.thick, Msg.src, Msg.sp)
			case '2':
				s.bspline(Msg.dst, Msg.pts, Msg.end0, Msg.end1, Msg.thick, Msg.src, Msg.sp)
			case 'P':
				s.poly(Msg.dst, Msg.pts, Msg.end0, Msg.end1, Msg.thick, Msg.src, Msg.sp)
			case 'L':
				s.line(Msg.dst, Msg.pt0, Msg.pt1, Msg.thick, Msg.src, Msg.sp)
			case 'd', 'x':
				if Msg.kind == 'd' {
					draw.Draw(Msg.dst, Msg.r, Msg.src, Msg.sp, Msg.op)
				} else {
					Msg.replyint <- s.stringBG(Msg.dst, Msg.p, Msg.src, Msg.sp, Msg.ft, Msg.data, Msg.bg, Msg.bgp)
					//s.drawBytes(msg.dst, msg.sp, msg.src, msg.data)
				}
			case 'f':
				r := Msg.r
				dp := s.sp.Add(r.Min)
				s.w.Upload(dp, s.b, r)
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

func (s *ShinyBitmap) stringBG(dst draw.Image, p image.Point, src image.Image, sp image.Point, ft *font.Font, data []byte, bg image.Image, bgp image.Point) int {
	return font.StringBG(dst, p, src, sp, ft, data, bg, bgp)
}

func (s *ShinyBitmap) StringBG(dst draw.Image, p image.Point, src image.Image, sp image.Point, ft *font.Font, data []byte, bg image.Image, bgp image.Point) int {
	replyint := make(chan int)
	s.ctl <- Msg{
		kind:     'x',
		dst:      dst,
		p:        p,
		src:      src,
		sp:       p,
		ft:       ft,
		data:     data,
		bg:       bg,
		bgp:      bgp,
		replyint: replyint,
	}
	return <-replyint
}

func (s *ShinyBitmap) Flush(r image.Rectangle) error {
	s.ctl <- Msg{
		kind: 'f',
		r:    r,
	}
	return nil
}
