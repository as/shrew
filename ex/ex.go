package main

import (
	"image"
	"image/color"
	"os"
	"time"
	//	"time"

	"github.com/as/frame"
	"github.com/as/frame/font"
	"github.com/as/shrew"
)

// Sends a message to the window system for a request to move. The window system
// send the request to the bitmap.
func FrameClient(c *shrew.Client) {
	r := image.Rectangle{image.ZP, c.W.Bounds().Size()}
	fr := frame.NewDrawer(r, font.NewGoMono(11), c.W, frame.A, c.W)
	tick := time.NewTicker(time.Second / 64)
	spin := byte(0)
	mp := image.ZP
	go func() {
		for m := range c.M {
			mp = m.Point
			i := fr.IndexOf(mp)
			fr.Select(i, i)
			c.W.Flush(image.ZR.Inset(-555))

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
		case <-tick.C:
			fr.Insert([]byte(mp.String()+"\n"), 0)
			spin++
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
			c.W.Draw(c.W, image.Rect(-1, -1, 1, 1).Inset(-2).Add(x), color, image.ZP, draw.Src)
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

func SolidClient(c *shrew.Client) {
	//col := rainbow
	//tick := time.NewTicker(time.Second / 2)
	pt0 := image.ZP
	pt0 = pt0
	var bpts [4]image.Point
	bn := 0
	bn = bn
	for {
		select {
		case m := <-c.M:
			m = m
			//c.W.Draw(c.W, c.W.Bounds(), rb, image.ZP, draw.Src)
			if m.Button == 1 {
			}
			if m.Button == 1<<1 {
				//c.W.Line(c.W, m.Point, pt0, 10, image.Black, image.ZP)
				//c.W.Line(c.W, pt0, m.Point, 1, image.NewUniform(color.RGBA{1,1,32,32}), image.ZP)
				copy(bpts[:], bpts[1:])
				bpts[3] = m.Point
				c.W.Bspline(c.W, bpts[:], 0, 0, 1, image.NewUniform(color.RGBA{255, 255, 255, 255}), image.ZP)
				c.W.Flush(c.W.Bounds())
			}
		case k := <-c.K:
			switch byte(k.Rune) {
			default:
				continue
			}
			//c.W.Draw(c.W, c.W.Bounds(), x, image.ZP, draw.Src)
			//c.W.Flush(c.W.Bounds())
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
		Bounds: image.Rect(0, 0, 1200, 800),
	}))
	//		go FrameClient(wsys.NewClient(&shrew.Options{
	//			Name: "frame",
	//			Bounds: image.Rect(500, 600, 1200, 800),
	//		}))
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
			if byte(k.Rune) == 't' {
				W.StringBG(W, image.Pt(100, 100), rb, image.ZP, ft, []byte(os.Args[0]), image.Black, image.ZP)
				W.Flush(image.ZR.Inset(-555))
			}
		case m := <-M:
			W.Flush(image.ZR.Inset(-5).Add(m.Point))
			//W = W
			m = m
			//		case <-tick.C:
			//			W.Draw(W, image.Rect(200, 200, 900, 300), image.White, image.ZP, draw.Src)
			//			tm := []byte(time.Now().String())
			//				W.StringBG(W, image.Pt(200, 200), rb, image.ZP, ft, tm, image.Black, image.ZP)
			//			W.Flush(image.ZR.Inset(-5555))
		default:
			next()
			rb = image.NewUniform(rainbow)
		}
	}
}
