package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/as/shrew"
)

/*
func FrameClient(c *shrew.Client) {
	fr := frame.New(c.W.Bounds(), font.NewGoMono(12), c.W.(*image.RGBA), frame.A)
	for {
		select {
		case m := <-c.M:
			i := fr.IndexOf(m.Point)
			fr.Select(i, i)
		case k := <-c.K:
			p0, _ := fr.Dot()
			fr.Insert([]byte{byte(k)}, p0)
			fr.Select(p0+1, p0+1)
		}
	}
}
*/
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
func SolidClient(c *shrew.Client) {
	r, g, b := image.NewUniform(color.RGBA{255, 0, 0, 255}), image.NewUniform(color.RGBA{0, 255, 0, 255}), image.NewUniform(color.RGBA{0, 0, 255, 255})
	x := image.White
	for {
		select {
		case <-c.M:
		case k := <-c.K:
			switch byte(k) {
			case byte('r'):
				x = r
			case byte('g'):
				x = g
			case byte('b'):
				x = b
			default:
				continue
			}
			c.W.Draw(c.W.Bounds(), x, image.ZP, draw.Src)
			c.W.Flush(c.W.Bounds())
		}
	}
}

func main() {
	wsys := shrew.NewWsys()
	//	go Spline(wsys.NewClient(&shrew.Options{
	//		Bounds: image.Rect(255, 255, 2560, 1440),
	//	}))
	go SolidClient(wsys.NewClient(&shrew.Options{
		Bounds: image.Rect(333, 333, 666, 666),
	}))
	//	go FrameClient(wsys.NewClient(&shrew.Options{
	//		Bounds: image.Rect(500, 600, 1200, 800),
	//	}))
	client := wsys.NewClient(&shrew.Options{
		Bounds: image.Rect(50, 50, 640, 480),
	})
	W, K, M := client.W, client.K, client.M
	var m shrew.Mouse
	for {
		select {
		case k := <-K:
			fmt.Printf("%#v\n", k)
			if byte(k) == 'q' {
				panic('q')
			}
		case m = <-M:
			fmt.Printf("%#v\n", m)
			W.Draw(image.ZR.Inset(-2).Add(m.Point), image.White, image.ZP, draw.Src)
			W.Flush(image.ZR.Inset(-2).Add(m.Point))
		}
	}
}
