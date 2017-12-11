package shrew

import (
	"image"
	"image/draw"

	"github.com/as/frame/font"
)

func (s *ShinyScreen) fontinit() (err error) {
	return err
}

func (s *ShinyBitmap) drawBytes(dst draw.Image, dot image.Point, src image.Image, b []byte) {
	ft := font.NewGoMedium(24)
	font.StringBG(dst, dot, src, image.ZP, ft, b, image.White, image.ZP)
}
