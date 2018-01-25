package shrew

import (
	"image"
	"image/draw"
	"sync"
)

func (w *wsys) merge(C ...<-chan Msg) <-chan Msg {
	var wg sync.WaitGroup
	out := make(chan Msg)

	output := func(c <-chan Msg) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}
	wg.Add(len(C))
	for _, c := range C {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

type wsys struct {
	// The window system has several independently executing clients, each of which has the same external
	// specification. It multiplexes a screen, mouse, and keyboard for its clients and therefore has a type reminiscent
	// of the clients themselves:
	W Screen

	M <-chan Mouse
	K <-chan Kbd
	C <-chan Msg
	N chan *Options

	Env     []*Env
	Nametab map[string]*Env
}

func NewWsys() Wsys {
	w := &wsys{
		N:       make(chan *Options),
		Nametab: make(map[string]*Env),
	}
	w.W = ShinyClient()
	w.M, w.K = w.W.Mouse(), w.W.Kbd()
	go w.prog()
	return w
}

func (w *wsys) newWindow(opt *Options) *Env {
	if !opt.Bounds.In(w.W.Bounds()) {
		println("bad bounds requested by client")
		close(opt.reply)
		return nil
	}
	bmp := w.W.AllocImage(opt.Bounds)
	e := &Env{
		Sp: opt.Bounds.Min,
		W:  bmp,
		M:  make(chan Mouse),
		K:  make(chan Kbd),
		C:  make(chan Msg),
	}
	w.Nametab[opt.Name] = e
	w.Env = append(w.Env, e)
	return e
}

func (w *wsys) prog() {
	var m Mouse
	for {
		select {
		case Msg := <-w.C:
			if Msg.Kind == "move" {
				w0 := w.Nametab[Msg.Name]
				type Mover interface {
					Move(sp image.Point)
				}
				//w.W.Draw(w.W, w0.W.Bounds(), BG, image.ZP, draw.Src)
				Msg.Sp = Msg.Sp.Add(w0.Sp)
				w0.W.(Mover).Move(Msg.Sp)
				w0.Sp = Msg.Sp
				drawBorder(w0.W, w0.W.Bounds(), image.Black, image.ZP, 1)
				w0.W.Flush(w0.W.Bounds())
			}
		case opt := <-w.N:
			e := w.newWindow(opt)
			if e == nil {
				continue
			}
			w.C = w.merge(w.C, e.C)
			opt.reply <- e
		case k := <-w.K:
			for _, c := range w.Env {
				r := c.W.Bounds().Add(c.Sp)
				if !m.Point.In(r) {
					continue
				}
				c.K <- k
			}
		case m = <-w.M:
			for _, c := range w.Env {
				r := c.W.Bounds().Add(c.Sp)
				if !m.Point.In(r) {
					continue
				}
				m := m
				m.Point = m.Point.Sub(r.Min)
				select {
				case c.M <- m:
				default:
				}
			}
		}
	}
}

func drawBorder(dst draw.Image, r image.Rectangle, src image.Image, sp image.Point, thick int) {
	draw.Draw(dst, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+thick), src, sp, draw.Src)
	draw.Draw(dst, image.Rect(r.Min.X, r.Max.Y-thick, r.Max.X, r.Max.Y), src, sp, draw.Src)
	draw.Draw(dst, image.Rect(r.Min.X, r.Min.Y, r.Min.X+thick, r.Max.Y), src, sp, draw.Src)
	draw.Draw(dst, image.Rect(r.Max.X-thick, r.Min.Y, r.Max.X, r.Max.Y), src, sp, draw.Src)
}

type Options struct {
	Name   string
	Bounds image.Rectangle
	reply  chan *Env
}

func (w *wsys) NewClient(opt *Options) Client {
	if opt == nil {
		opt = &Options{
			Name:   "unnamed",
			Bounds: image.Rect(0, 0, 1, 1),
		}
	}
	opt.reply = make(chan *Env)
	w.N <- opt
	e := <-opt.reply
	if e == nil {
		return nil
	}
	return &client{
		name:   opt.Name,
		Bitmap: e.W,
		m:      e.M,
		k:      e.K,
		c:      e.C,
	}
}
