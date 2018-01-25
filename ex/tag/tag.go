package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime/pprof"
	"strings"
	"sync"
	//	"time"
	_ "image/jpeg"

	"github.com/as/frame"
	"github.com/as/shrew"
	. "github.com/as/ui/win"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func trypprof() func() {
	flag.Parse()
	if *cpuprofile == "" {
		return func() {}
	}
	f, err := os.Create(*cpuprofile)
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	return pprof.StopCPUProfile
}
func BeizerClient(c shrew.Client) {

}

type WinConfig struct {
	*Config
	ConIn, ConOut chan []byte
	TX            chan wintx
}

func (w *WinConfig) check() {
	if w.ConIn == nil {
		w.ConIn = make(chan []byte)
	}
	if w.ConOut == nil {
		w.ConOut = make(chan []byte)
	}
	if w.TX == nil {
		w.TX = make(chan wintx)
	}
}

func WinClient(c shrew.Client, conf *WinConfig) {
	conf.check()
	var (
		pts     [2]image.Point
		aux     = conf.TX
		conin   = conf.ConIn
		conout  = conf.ConOut
		insertC = make(chan []byte)
		//		appendC   = make(chan []byte)
		deleteC   = make(chan [2]int)
		selectPTC = make(chan [2]image.Point)
		selectC   = make(chan [2]int)
		scrollC   = make(chan int)
		vsyncC    = make(chan struct{})
	)
	go cmdexec(conin, conout)
	q2 := int64(0)
	w := New(c, conf.Config)
	w.Flush(c.Bounds())
	var (
		kbd = c.K()
		mus = c.M()
	)
	go func() {
		var (
			m shrew.Mouse
			i int
		)
		for m = range mus {
			switch m.Button {
			case 1 << 4:
				scrollC <- 5
			case 1 << 3:
				scrollC <- -5
			case 0:
				i = 0
			case 1 << 2:
				for m2 := range mus {
					if m2.Button != 1<<2 {
						break
					}
					c.C() <- shrew.Msg{
						Sp:   m2.Point.Sub(m.Point),
						Name: c.Name(),
						Kind: "move",
					}
				}
			case 1:
				pts[0] = m.Point
				if i == 0 {
					pts[1] = pts[0]
					i++
				}
				selectPTC <- pts
			}
		}
	}()
	go func() {
		for k := range kbd {
			if k.Press == 0 || k.Rune == '\xff' {
				continue
			}
			if k.Rune == 'Q' {
				pprof.StopCPUProfile()
				os.Exit(0)
			}
			insertC <- []byte{byte(k.Rune)}
		}
	}()
	for {
		w.Buffer()
		select {
		case dy := <-scrollC:
		Loop:
			for n := 1; ; n++ {
				select {
				case y := <-scrollC:
					dy += y * n
				default:
					w.Scroll(dy)
					break Loop
				}
			}
		case p := <-conout:
			q2 += int64(w.Insert(p, q2))
			q0, _ := w.Dot()
			if q0 >= q2 {
				w.SetOrigin(w.Len()-w.Frame.Len(), true)
			}
		case p := <-insertC:
			q0, q1 := w.Dot()
			var skip bool
			if p[0] == '\x08' {
				if q0 != 0 {
					q0--
				}
				skip = true
			}
			if q0 != q1 {
				n := w.Delete(q0, q1)
				if q2 <= q1 && q2 > q0 {
					if q2 <= q1 {
						q2 = q0
					} else {
						q2 -= int64(n)
					}
				}
			}
			if skip {
				break
			}
			if p[0] == '\n' && q2 < q0 {
				str := w.Bytes()[q2:q0]
				q2++
				if len(str) > 1 && str[len(str)-1] == '\n' {
					str = str[:len(str)-1]
					q2++
				}
				q2 += int64(len(str))
				conin <- append([]byte{}, str...)
			} else if q2 > q0 {
				q2 += int64(len(p))
			}
			n := w.Insert(p, q0)
			q0 += int64(n)
			w.Select(q0, q0)
			select {
			default:
			case aux <- wintx{{kind: 'x', p: append([]byte{}, w.Bytes()...)}}:
			}
		case q := <-deleteC:
			q0, q1 := int64(q[0]), int64(q[1])
			n := w.Delete(q0, q1)
			if q2 <= q1 && q2 > q0 {
				if q2 <= q1 {
					q2 = q0
				} else {
					q2 -= int64(n)
				}
			}
		case pt := <-selectPTC:
			org := w.Origin()
			w.Select(int64(w.IndexOf(pt[0]))+org, int64(w.IndexOf(pt[1]))+org)
		case q := <-selectC:
			org := w.Origin()
			w.Select(int64(q[0])+org, int64(q[1])+org)
		case <-vsyncC:
			w.Unbuffer()
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

func main() {
	defer trypprof()()
	auximg := make(chan wintx)
	wsys := shrew.NewWsys()
	go WinClient(wsys.NewClient(&shrew.Options{
		Name:   "tag0",
		Bounds: image.Rect(0, 0, 1900, 30),
	}), &WinConfig{
		Config: &Config{
			Face:  frame.StaticFace(frame.NewGoMono(11)),
			Pad:   image.Pt(15, 5),
			Color: &frame.ATag0,
		},
	})
	go WinClient(wsys.NewClient(&shrew.Options{
		Name:   "tag1",
		Bounds: image.Rect(0, 30, 1900, 60),
	}), &WinConfig{
		TX: auximg,
		Config: &Config{
			Face:  frame.StaticFace(frame.NewGoMono(11)),
			Pad:   image.Pt(15, 5),
			Color: &frame.ATag1,
		},
	})
	n := 2
	x0, dx := 0, 1900/n
	for i := 0; i < n; i++ {
		go WinClient(wsys.NewClient(&shrew.Options{
			Name:   "frame" + fmt.Sprint(i),
			Bounds: image.Rect(x0, 60, x0+dx, 600),
		}), &WinConfig{
			Config: &Config{
				Face:  frame.StaticFace(frame.NewGoMono(11)),
				Color: &frame.A,
			},
		})
		x0 += dx
	}
	go WinClient(wsys.NewClient(&shrew.Options{
		Name:   "debug",
		Bounds: image.Rect(0, 1000, 1900, 1100),
	}), &WinConfig{
		Config: &Config{
			Face:  frame.StaticFace(frame.NewGoMono(11)),
			Color: &frame.ATag0,
		},
	})
	c := wsys.NewClient(&shrew.Options{
		Name:   "image",
		Bounds: image.Rect(0, 600, 1900, 1000),
	})
	mus, kbd := c.M(), c.K()
	for {
		select {
		case m := <-auximg:
			name := string(m[0].p)
			fd, err := os.Open(name)
			if err != nil {
				continue
			}
			hi, _, err := image.Decode(fd)
			if err != nil {
				continue
			}
			c.Draw(c, c.Bounds(), hi, hi.Bounds().Min, draw.Src)
			c.Flush(hi.Bounds())
		case <-mus:
		case k := <-kbd:
			if k.Rune == 'Q' {
				break
			}
		}
	}
}

func cmdexec(conin, conout chan []byte) {
	for {
		select {
		case p := <-conin:
			x := strings.Fields(string(p))
			if len(x) == 0 {
				continue
			}
			n := x[0]
			var a []string
			if len(x) > 1 {
				a = x[1:]
			}
			cmd := exec.Command(n, a...)
			fd0, _ := cmd.StdinPipe()
			fd1, _ := cmd.StdoutPipe()
			fd2, _ := cmd.StderrPipe()
			var wg sync.WaitGroup
			donec := make(chan bool)
			wg.Add(2)
			startfd := func(fd io.Reader) {
				defer wg.Done()
				//defer fd.Close()
				b := make([]byte, 64*1024)
				for {
					select {
					case <-donec:
						return
					default:
						n, err := fd.Read(b)
						if n > 0 {
							conout <- append([]byte{}, b[:n]...)
						}
						if err != nil {
							return
						}
					}
				}
			}
			go startfd(bufio.NewReader(fd1))
			go startfd(bufio.NewReader(fd2))
			err := cmd.Start()
			if err != nil {
				println(err)
			}
			go func() {
				wg.Wait()
				close(donec)
				cmd.Wait()
			}()
		Loop:
			for {
				select {
				case p := <-conin:
					fd0.Write(append(p, '\n'))
				case <-donec:
					break Loop
				}
			}
		}
	}
}

/*
	case msgs := <-wintxC:
		for _, v := range msgs {
			q := v.q
			switch v.kind {
			case '$':
				w.Scroll(int(v.q[0]))
			case 'a':
				q := q2 - 1
				if q < 0 {
					q++
				}
				n := w.Insert(v.p, q2)
				q0, q1 := w.Dot()
				if q2 < q0 {
					q0 += int64(n)
					q1 += int64(n)
					w.Select(q0, q1)
					q2 += int64(n)
				}
				// scroll?
				if w.Origin()+w.Frame.Len() < q2 {
					w.SetOrigin(q2-w.Frame.Len(), false)
				}
			case 'i':
				q0, q1 := w.Dot()
				s := v.p
				var skip bool
				if s[0] == '\x08' {
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
				if s[0] == '\n' && q2 < q0 {
					str := w.Bytes()[q2:q0]
					q2++
					if len(str) > 1 && str[len(str)-1] == '\n' {
						str = str[:len(str)-1]
						q2++
					}
					conin <- str
					q2 += int64(len(str))
				} else if q2 > q0 {
					q2 += int64(len(s))
				}
				n := w.Insert(s, q0)
				q0 += int64(n)
				w.Select(q0, q0)
				select {
				default:
				case aux <- wintx{{kind: 'x', p: append([]byte{}, w.Bytes()...)}}:
				}
			case 'd':
				w.Delete(int64(q[0]), int64(q[1]))
			case 'D':
			case 'S':
				q[0] = w.IndexOf(v.pts[0])
				q[1] = w.IndexOf(v.pts[1])
				fallthrough
			case 's':
				org := w.Origin()
				w.Select(int64(q[0])+org, int64(q[1])+org)
			case 'v':
				w.Unbuffer()
			}
*/
