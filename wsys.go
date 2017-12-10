package shrew

import (
	"fmt"
	"image"
	"image/draw"
	"log"
	"sync"
	"time"

	"github.com/as/ui"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
)

type Wsys struct {
	// The window system has several independently executing clients, each of which has the same external
	// specification. It multiplexes a screen, mouse, and keyboard for its clients and therefore has a type reminiscent
	// of the clients themselves:
	W Screen

	M <-chan Mouse
	K <-chan Kbd
	N chan *Options

	Env []*Env
}

func NewWsys() *Wsys {
	w := &Wsys{
		N: make(chan *Options),
	}
	w.W = ShinyClient()
	w.M, w.K = w.W.Mouse(), w.W.Kbd()
	go w.prog()
	return w
}

func (w *Wsys) newWindow(opt *Options) *Env {
	if !opt.Bounds.In(w.W.Bounds()) {
		println("bad bounds requested by client")
		close(opt.reply)
		return nil
	}
	fmt.Println(opt.Bounds)
	bmp := w.W.AllocImage(opt.Bounds)
	e := &Env{
		W: bmp,
		M: make(chan Mouse),
		K: make(chan Kbd),
	}
	w.Env = append(w.Env, e)
	return e
}

func (w *Wsys) prog() {
	for {
		select {
		//		case msg := <-w.CI:
		// always v and from client 1
		// a draw message that says "refresh"

		case opt := <-w.N:
			e := w.newWindow(opt)
			if e == nil {
				continue
			}
			opt.reply <- e
		case k := <-w.K:
			for _, c := range w.Env {
				c.K <- k
			}
		case m := <-w.M:
			for i, c := range w.Env {
				select {
				case c.M <- m:
				case <-time.After(time.Second):
					log.Printf("client %d is slow\n", i)
				}
			}
		}
	}
}

type Options struct {
	Bounds image.Rectangle
	reply  chan *Env
}

func (w *Wsys) NewClient(opt *Options) *Client {
	if opt == nil {
		opt = &Options{
			Bounds: image.Rect(0, 0, 1, 1),
		}
	}
	opt.reply = make(chan *Env)
	w.N <- opt
	e := <-opt.reply
	if e == nil {
		return nil
	}
	return &Client{
		W: e.W,
		M: e.M,
		K: e.K,
	}
}

//func allocimage(d Display, r image.Rectangle) draw.Image{
//	return shinyallocimage(r)
//}

//func shinyallocimage(r image.Rectangle) draw.Image{
///	return
//}

var dev *ui.Dev

type ShinyBitmap struct {
	sp      image.Point
	size    image.Point
	b       screen.Buffer
	w       screen.Window
	ctl     chan msg
	wg      sync.WaitGroup
	refresh chan msg
	draw    chan msg
}

type msg struct {
	kind   byte
	r      image.Rectangle
	src    image.Image
	sp     image.Point
	op     draw.Op
	replyc chan error
}

func (s *ShinyBitmap) run() {
	s.refresh = make(chan msg)
	s.draw = make(chan msg)
	s.ctl = make(chan msg)
	go func() {
		for v := range s.ctl {
			if v.kind == 'd' {
				s.draw <- v
			} else if v.kind == 'f' {
				s.refresh <- v
			}
			log.Println("invalid msg", v)
		}
	}()
	for {
		select {
		case msg := <-s.refresh:
			s.wg.Wait()
			dp := msg.r.Min
			r := msg.r.Sub(s.Bounds().Min)
			s.w.Upload(dp, s.b, r)
			close(msg.replyc)
		case msg := <-s.draw:
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				r := msg.r.Sub(s.Bounds().Min)
				draw.Draw(s.b.RGBA(), r, msg.src, msg.sp, msg.op)
			}()
		}
	}
}

func (s *ShinyBitmap) Bounds() image.Rectangle {
	return s.b.Bounds().Add(s.sp)
}
func (s *ShinyBitmap) Draw(r image.Rectangle, src image.Image, sp image.Point, op draw.Op) {
	s.ctl <- msg{
		kind: 'd',
		r:    r,
		src:  src,
		sp:   sp,
		op:   op,
	}
}
func (s *ShinyBitmap) Flush(r image.Rectangle) error {
	ch := make(chan error)
	s.ctl <- msg{
		kind:   'f',
		r:      r,
		replyc: ch,
	}
	return <-ch
}

func (s *ShinyScreen) AllocImage(r image.Rectangle) Bitmap {
	b := s.dev.NewBuffer(r.Size())
	bmp := &ShinyBitmap{
		sp:   r.Min,
		size: r.Size(),
		b:    b,
		w:    s.dev.Window(),
	}
	go bmp.run()
	return bmp
}
func (s *ShinyScreen) Kbd() chan Kbd           { return s.K }
func (s *ShinyScreen) Mouse() chan Mouse       { return s.M }
func (s *ShinyScreen) Bounds() image.Rectangle { return image.Rect(0, 0, 2500, 1400) }

type ShinyScreen struct {
	dev *ui.Dev
	K   chan Kbd
	M   chan Mouse
}

func ShinyClient() *ShinyScreen {
	dev, err := ui.Init(nil)
	println(err)
	w := dev.Window()
	K := make(chan Kbd)
	M := make(chan Mouse)
	sc := &ShinyScreen{
		dev: dev,
		K:   K,
		M:   M,
	}
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
				K <- Kbd(e.Rune)
			case paint.Event:
			case interface{}:
				println("unknown event")
			}
		}
	}()
	return sc
}
