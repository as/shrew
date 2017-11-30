package shrew

import (
	"image"
	"log"
	"time"

	"github.com/as/ui"
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
	w.W, w.K, w.M = ShinyClient()
	go w.prog()
	return w
}

func (w *Wsys) newWindow(opt *Options) *Env {
	if !opt.Bounds.In(w.W.Bounds()) {
		println("bad bounds requested by client")
		close(opt.reply)
		return nil
	}
	e := &Env{
		W: w.W.SubImage(opt.Bounds).(Bitmap),
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

func ShinyClient() (Bitmap, <-chan Kbd, <-chan Mouse) {
	dev, err := ui.Init(nil)
	println(err)
	w := dev.Window()
	b := dev.NewBuffer(image.Pt(2560, 1440))
	K := make(chan Kbd)
	M := make(chan Mouse)
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
				w.Upload(b.Bounds().Min, b, b.Bounds())
				w.Send(paint.Event{})
			case interface{}:
				println("unknown event")
			}
		}
	}()
	return Bitmap(b.RGBA()), K, M
}
