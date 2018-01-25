package main

import (
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
				go func() {
					cmd := exec.Command(n, a...)
					fd0, _ := cmd.StdinPipe()
					fd1, _ := cmd.StdoutPipe()
					fd2, _ := cmd.StderrPipe()
					var wg sync.WaitGroup
					donec := make(chan bool)
					wg.Add(2)
					startfd := func(fd io.ReadCloser) {
						defer wg.Done()
						b := make([]byte, 65536)
						for {
							select {
							case <-donec:
								return
							default:
								n, err := fd.Read(b)
								if err != nil {
									if err == io.EOF {
										break
									}
								}
								conout <- append([]byte{}, b[:n]...)
							}
						}
					}
					startfd(fd1)
					startfd(fd2)
					cmd.Start()
				Loop:
					for {
						select {
						case p := <-conin:
							fd0.Write(p)
						case <-donec:
							break Loop
						}
					}
					cmd.Wait()
				}()
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
			case 'a':
				w.Insert(v.p, q2)
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
				if s == "\n" {
					str := string(w.Bytes()[q2:q0])
					conin <- []byte(str)
					q2 = q0 - 1
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

func main() {
	wsys := shrew.NewWsys()
	WinClient(wsys.NewClient(&shrew.Options{
		Name:   "frame",
		Bounds: image.Rect(0, 0, 1900, 1000),
	}))
}
