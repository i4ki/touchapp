// +build darwin linux windows
package main

import (
	"image"
	"image/draw"

	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/exp/gl/glutil"
	"golang.org/x/mobile/geom"
)

type Bunny struct {
	sz     size.Event
	images *glutil.Images
	src    image.Image
	m      *glutil.Image
	// TODO: store *gl.Context
}

// NewBunny creates an bunny image
func NewBunny(images *glutil.Images, src image.Image) *Bunny {
	return &Bunny{
		images: images,
		src:    src,
	}
}

func (p *Bunny) Draw(sz size.Event, x, y float32) {
	imgW, imgH := geom.Pt(p.src.Bounds().Dx()), geom.Pt(p.src.Bounds().Dy())

	if sz.WidthPx == 0 && sz.HeightPx == 0 {
		return
	}
	if p.sz != sz {
		p.sz = sz
		if p.m != nil {
			p.m.Release()
		}
		p.m = p.images.NewImage(int(imgW), int(imgH))
	}

	draw.Draw(p.m.RGBA, p.m.RGBA.Bounds(), p.src, p.src.Bounds().Min, draw.Src)
	p.m.Upload()

	p.m.Draw(
		sz,
		geom.Point{geom.Pt(x) - imgW/2, geom.Pt(y) - imgH/2},
		geom.Point{geom.Pt(x) - imgW/2 + imgW, geom.Pt(y) - imgH/2},
		geom.Point{geom.Pt(x) - imgW/2, geom.Pt(y) - imgH/2 + imgH},
		p.m.RGBA.Bounds(),
	)
}

func (f *Bunny) Release() {
	if f.m != nil {
		f.m.Release()
		f.m = nil
		f.images = nil
	}
}
