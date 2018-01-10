package shrew

import (
	"image"
	"log"
	"sync"
	"time"
)

func (w *Wsys) merge(C ...<-chan Msg) <-chan Msg {
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

type Wsys struct {
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

func NewWsys() *Wsys {
	w := &Wsys{
		N:       make(chan *Options),
		Nametab: make(map[string]*Env),
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
	bmp := w.W.AllocImage(opt.Bounds)
	e := &Env{
		Sp: opt.Bounds.Min,
		W:  bmp,
		M:  make(chan Mouse, 50),
		K:  make(chan Kbd),
		C:  make(chan Msg),
	}
	w.Nametab[opt.Name] = e
	w.Env = append(w.Env, e)
	return e
}

func (w *Wsys) prog() {
	for {
		select {
		case Msg := <-w.C:
			if Msg.Kind == "move" {
				w := w.Nametab[Msg.Name]
				type Mover interface {
					Move(sp image.Point)
				}
				Msg.Sp = Msg.Sp.Add(w.Sp)
				w.W.(Mover).Move(Msg.Sp)
				w.Sp = Msg.Sp
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
				c.K <- k
			}
		case m := <-w.M:
			for i, c := range w.Env {
				m := m
				m.Point = m.Point.Sub(c.Sp)
				select {
				case c.M <- m:
					if cap(c.M)/4 < len(c.M) {
						for cap(c.M)/7 < len(c.M) {
							<-c.M
						}
					}
				case <-time.After(time.Second / 32):
					log.Printf("client %d is slow\n", i)
					if cap(c.M)/4 < len(c.M) {
						for cap(c.M)/7 < len(c.M) {
							<-c.M
						}
					}
				}
			}
		}
	}
}

type Options struct {
	Name   string
	Bounds image.Rectangle
	reply  chan *Env
}

func (w *Wsys) NewClient(opt *Options) *Client {
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
	return &Client{
		W: e.W,
		M: e.M,
		K: e.K,
		C: e.C,
	}
}
