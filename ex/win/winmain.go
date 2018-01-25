package main

import (
	"bufio"
	"image"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"
	//	"time"

	"github.com/as/shrew"
)

var (
	wintxC  = make(chan wintx)
	updateC = make(chan []byte)
	conin   = make(chan []byte)
	conout  = make(chan []byte)
)

func WinClient(c *shrew.client) {
	q2 := int64(0)
	w := New(c, nil)
	w.Flush(c.W.Bounds())
	go func() {
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
					b := make([]byte, 32768)
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
	}()
	go func() {
		var (
			m        shrew.Mouse
			pts      = make([]image.Point, 2)
			i        int
			lastdata string
		)
		for m = range c.M {
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
			switch m.Button {
			case 1 << 3, 1 << 4:
				q0 := int64(3)
				if m.Button == 1<<4 {
					q0 = -q0
				}
				wintxC <- wintx{
					{kind: '$', q: [2]int64{q0, q0}},
				}
			case 0:
				i = 0
			case 1:
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
	go func() {
		for k := range c.K {
			if k.Press == 0 || k.Rune == '\xff' {
				continue
			}
			if k.Rune == 'q' {
				panic("q")
			}
			wintxC <- wintx{
				{kind: 'i', p: []byte{byte(k.Rune)}},
			}
		}
	}()
	go func() {
		for p := range conout {
			wintxC <- wintx{
				{kind: 'a', p: p},
			}
		}
	}()
	for msgs := range wintxC {
		w.Buffer()
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
				s := string(v.p)
				var skip bool
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
				if s == "\n" && q2 < q0 {
					str := string(w.Bytes()[q2:q0])
					q2++
					if len(str) > 1 && str[len(str)-1] == '\n' {
						str = str[:len(str)-1]
						q2++
					}
					conin <- []byte(str)
					q2 += int64(len(str))
				} else if q2 > q0 {
					q2 += int64(len(s))
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
				org := w.Origin()
				w.Select(int64(q[0])+org, int64(q[1])+org)
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

func main() {
	wsys := shrew.NewWsys()
	WinClient(wsys.NewClient(&shrew.Options{
		Name:   "frame",
		Bounds: image.Rect(0, 0, 1900, 1000),
	}))
}
