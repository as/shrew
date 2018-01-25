package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"time"
	//	"time"

	"github.com/as/frame"
	"github.com/as/frame/font"
	"github.com/as/shrew"
	. "github.com/as/shrew/win"
)

var (
	wintxC  = make(chan wintx)
	updateC = make(chan []byte)
)

func WinClient(c shrew.Client) {
	w := New(c, nil)
	go func() {
		var (
			m        shrew.Mouse
			pts      = make([]image.Point, 2)
			i        int
			lastdata string
		)
		for m = range c.M() {
			if !m.Point.In(c.W.Bounds()) {
				data := string(w.Bytes())
				if data != lastdata {
					updateC <- []byte(data)
					lastdata = data
				}
				for m = range c.M {
					if m.Point.In(c.W.Bounds()) {
						break
					}
				}
			}
			switch {
			case m.Button == 0:
				i = 0
			case m.Button == 1:
				pts[0] = m.Point
				if i == 0 {
					pts[1] = pts[0]
					i++
				}
				wintxC <- wintx{
					{kind: 'S', pts: append([]image.Point{}, pts...)},
				}
			}
		}
	}()
	w.Flush(c.W.Bounds())
	go func() {
		for k := range c.K {
			if k.Press == 0 {
				continue
			}
			wintxC <- wintx{
				{kind: 'i', p: []byte{byte(k.Rune)}},
			}
		}
	}()
	for msgs := range wintxC {
		w.Buffer()
		for _, v := range msgs {
			q := v.q
			switch v.kind {
			case 'i':
				s := string(v.p)
				var skip bool
				q0, q1 := w.Dot()
				if s == "\x08" {
					if q0 != 0 {
						q0--
					}
					skip = true
				}
				if q0 != q1 {
					w.Delete(q0, q1)
				}
				if skip {
					break
				}
				n := w.Insert([]byte(s), q0)
				q0 += int64(n)
				w.Select(q0, q0)
			case 'd':
				w.Delete(int64(q[0]), int64(q[1]))
			case 'D':
			case 'S':
				q[0] = w.IndexOf(v.pts[0])
				q[1] = w.IndexOf(v.pts[1])
				fallthrough
			case 's':
				w.Select(int64(q[0]), int64(q[1]))
			case 'v':
				w.Unbuffer()
			}
		}
		w.Unbuffer()
	}
}

type wintx []winmsg

type winmsg struct {
	kind byte
	q    [2]int64
	pts  []image.Point
	p    []byte
}

// Sends a message to the window system for a request to move. The window system
// send the request to the bitmap.
// haven't updated the frame package yet...
func FrameClient(c shrew.client) {
	r := image.Rectangle{image.ZP, c.W.Bounds().Size()}
	fr := frame.NewDrawer(r.Inset(5), font.NewGoMono(12), c.W, frame.A, c.W)
	mp := image.ZP
	go func() {
		for m := range c.M {
			mp = m.Point
			i := fr.IndexOf(mp)
			fr.Select(i, i)
			if m.Button == 1 {
				start := m.Point
				for m.Button == 1 {
					c.C <- shrew.Msg{
						Name: "frame",
						Kind: "move",
						Sp:   m.Point.Sub(start),
					}
					m = <-c.M
					c.W.Flush(image.ZR.Inset(-555))
				}
			}
		}
	}()
	for {
		select {
		case k := <-c.K:
			if k.Press == 0 {
				continue
			}
			p0, _ := fr.Dot()
			fr.Insert([]byte{byte(k.Rune)}, p0)
			fr.Select(p0+1, p0+1)
			c.W.Flush(image.ZR.Inset(-555))
		}
	}
}

/*
func Spline(c *shrew.Client) {
	r := image.NewUniform(color.RGBA{255, 0, 0, 255})
	y := image.NewUniform(color.RGBA{255, 255, 0, 255})
	b := image.Black
	p := make([]image.Point, 4)
	fun := memdraw.Bezspline
	bsp := func(color image.Image) {
		fun(c.W, p[0], p[1], p[2], p[3], 1, 1, 4, color, image.ZP)
		cum := image.ZR
		for _, x := range p {
			if color != b {
				color = y
			}
			if false{
			c.W.Draw(c.W, image.Rect(-1, -1, 1, 1).Inset(-2).Add(x), color, image.ZP, draw.Src)
			}
			cum := cum.Union(image.Rect(-1, -1, 1, 1).Inset(-2).Add(x))
			//c.W.Flush(image.Rect(-1, -1, 1, 1).Inset(-2).Add(x))
			//memdraw.Line(c.W, p[0], p[3], 10, image.White, image.ZP)
		}
		c.W.Flush(cum)
	}
	for {
		select {
		case m := <-c.M:
			if m.Button == 1 {
				bsp(b)
				p = append(p, m.Point)
				p = p[1:]

				bsp(r)
				for m := range c.M {
					if m.Button != 1 {
						break
					}
				}
			}
			//bsp(b)
			//for i := range p{
			//p[i] = p[i].Add(image.Pt(rand.IntN(100), rand.IntN(100))
			//}
			//bsp(r)
		case k := <-c.K:
			switch byte(k) {
			case byte('b'):
				fun = memdraw.Bezier
			case byte('s'):
				fun = memdraw.Bezspline
			default:
				continue
			}
			//draw.Draw(c.W, c.W.Bounds(), x, image.ZP, draw.Src)
		}
	}
}
*/

var rb = image.NewUniform(rainbow)

type P struct {
	X, Y float64
}

func (p *P) Point() image.Point {
	return image.Pt(int(p.X+0.5), int(p.Y+0.5))
}
func SolidClient(c shrew.client) {
	//col := rainbow
	//tick := time.NewTicker(time.Second / 2)
	pt0 := image.ZP
	pt0 = pt0
	var bpts [9]image.Point
	bn := 0
	bn = bn
	var curvefn func(t float64, p []P) P
	curvefn = func(t float64, p []P) P {
		if len(p) == 1 {
			return p[0]
		}
		//for i := 0;i<125;i++{
		//next()
		//	}
		rb = image.NewUniform(rainbow)
		p2 := make([]P, 0, len(p)-1)
		for i := 0; i < len(p)-1; i++ {
			fx0 := float64(p[i].X)
			fy0 := float64(p[i].Y)
			fx1 := float64(p[i+1].X)
			fy1 := float64(p[i+1].Y)
			x := (1.0-t)*fx0 + t*fx1
			y := (1.0-t)*fy0 + t*fy1
			//r := image.Rect(-1, -1, 1, 1).Add(image.Pt(int(x), int(y)))
			//c.W.Draw(c.W, r, rb, image.ZP, draw.Src)
			//c.W.Flush(c.W.Bounds())
			p2 = append(p2, P{x, y})
		}
		return curvefn(t, p2)
	}
	c.W.Line(c.W, image.Pt(10, 10), image.Pt(13, 17), 10, image.Black, image.ZP)
	c.W.Line(c.W, image.Pt(50+10, 50+10), image.Pt(50+17, 50+13), 10, image.White, image.ZP)
	var m shrew.Mouse
	tick := time.NewTicker(time.Second / 30)
	zts := make([]P, 0, len(bpts))
	qts := make([]P, 0, len(bpts))
	go func() {
		for range c.K {
		}
	}()
	parsept := func(b []byte) {
		var i int
		var x, y int
		if len(b) < 3 {
			return
		}
		br := bytes.NewReader(b)
		for n := 0; n < len(bpts); n++ {
			_, err := fmt.Fscanf(br, "#%d: (%d,%d)\n", &i, &x, &y)
			if err != nil {
				break
			}
			bpts[i] = image.Pt(x, y)
		}
	}
	genpt := func() {
		c.W.Draw(c.W, c.W.Bounds(), image.Black, image.ZP, draw.Src)
		const (
			seg  = 1000
			step = 1.0 / seg
		)
		for i, q := range bpts[:] {
			c.W.Draw(c.W, image.ZR.Inset(-(i + 1)).Add(q), image.White, image.ZP, draw.Src)
		}
		zts = zts[:0]

		wtx := wintx{
			{kind: 'd', q: [2]int64{0, 999}},
		}
		for i := range bpts {
			msg := fmt.Sprintf("#%d: %s\n", i, bpts[i])
			wtx = append(wtx, winmsg{kind: 'i', p: []byte(msg)})
			zts = append(zts, P{float64(bpts[i].X), float64(bpts[i].Y)})
		}
		wintxC <- wtx

		qts = qts[:1]
		qts[0] = zts[0]
		t := 0.0
		for i := 0; i < seg; i++ {
			qts = append(qts, curvefn(t, zts[:]))
			t += step
		}
		qts = append(qts, zts[len(zts)-1])
		for i := range qts[:len(qts)-1] {
			c.W.Line(c.W, qts[i].Point(), qts[i+1].Point(), 1, image.NewUniform(color.RGBA{255, 255, 255, 255}), image.ZP)
			//c.W.Flush(c.W.Bounds())
		}
		c.W.Flush(c.W.Bounds())
	}

	for i := 0; ; i = i {
		select {
		case b := <-updateC:
			parsept(b)
			genpt()
		case m = <-c.M:
			if m.Button == 1 {
				i := 0
				for ; i < len(bpts); i++ {
					if bpts[i].In(image.ZR.Inset(-10).Add(m.Point)) {
						break
					}
				}
				if i == len(bpts) {
					continue
				}
				for m.Button == 1 {
					bpts[i] = m.Point
					m = <-c.M
					genpt()
				}
			}
			if !m.Point.In(c.W.Bounds()) {
				continue
			}
			if m.Button != 1<<2 {
				continue
			}
			copy(bpts[:], bpts[1:])
			bpts[len(bpts)-1] = m.Point
			select {
			case <-tick.C:
				genpt()
			default:
			}
		}
	}
}

var rainbow = color.RGBA{255, 0, 0, 255}

func next() {
	rainbow = nextcolor(rainbow)
}

// nextcolor steps through a gradient
func nextcolor(c color.RGBA) color.RGBA {
	switch {
	case c.R == 255 && c.G == 0 && c.B == 0:
		c.G += 25
	case c.R == 255 && c.G != 255 && c.B == 0:
		c.G += 25
	case c.G == 255 && c.R != 0:
		c.R -= 25
	case c.R == 0 && c.B != 255:
		c.B += 25
	case c.B == 255 && c.G != 0:
		c.G -= 25
	case c.G == 0 && c.R != 255:
		c.R += 25
	default:
		c.B -= 25
	}
	return c
}

func main() {
	wsys := shrew.NewWsys()
	//	go Spline(wsys.NewClient(&shrew.Options{
	//			Bounds: image.Rect(255, 255, 2560, 1440),
	//		}))
	go SolidClient(wsys.NewClient(&shrew.Options{
		Bounds: image.Rect(0, 0, 1024, 768),
	}))
	go WinClient(wsys.NewClient(&shrew.Options{
		Name:   "frame",
		Bounds: image.Rect(0, 768, 1024, 1024),
	}))
	client := wsys.NewClient(&shrew.Options{
		Bounds: image.Rect(1, 1, 500, 500),
	})
	W, K, M := client.W, client.K, client.M
	//	tick := time.NewTicker(time.Second / 64)
	ft := font.NewGoMono(33)
	for {
		select {
		case k := <-K:
			if byte(k.Rune) == 'q' {
				panic('q')
			}
			continue
			if byte(k.Rune) == 't' {
				W.StringBG(W, image.Pt(100, 100), rb, image.ZP, ft, []byte(os.Args[0]), image.Black, image.ZP)
				W.Flush(image.ZR.Inset(-555))
			}
		case m := <-M:
			continue
			W.Flush(image.ZR.Inset(-5).Add(m.Point))
			//W = W
			m = m
			//		case <-tick.C:
			//			W.Draw(W, image.Rect(200, 200, 900, 300), image.White, image.ZP, draw.Src)
			//			tm := []byte(time.Now().String())
			//				W.StringBG(W, image.Pt(200, 200), rb, image.ZP, ft, tm, image.Black, image.ZP)
			//			W.Flush(image.ZR.Inset(-5555))
		}
	}
}
