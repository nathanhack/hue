package gui

import (
	"fmt"
	"image/color"
	"math"
	"net"
	"sync"
	"time"

	"github.com/golang/freetype/truetype"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/text"
	"golang.org/x/image/font"
)

const (
	dpi        = 72
	FadingTime = 300 * time.Millisecond
)

var (
	blue       = color.RGBA{0, 168, 255, 255}
	grey       = color.RGBA{138, 138, 138, 255}
	black      = color.RGBA{0, 0, 0, 255}
	red        = color.RGBA{0xb0, 0, 0x20, 255}
	normalFont = make(map[int]font.Face)
	tt         *truetype.Font
)

type Fade int

const (
	OffAppearing Fade = iota
	OnAppearing
	OnFading
	OffFading
)

type guiLight struct {
	label             string
	x, y              float32
	size              float32
	simple            bool
	on                bool
	num               int
	drawing           sync.Mutex
	onVertices        [][]ebiten.Vertex
	offVertices       [][]ebiten.Vertex
	redVertices       [][]ebiten.Vertex
	indices           [][]uint16
	err               bool
	lastSeen          time.Time
	lastAddressUpdate time.Time
	address           *net.UDPAddr
	lastChange        time.Time
	fadeState         Fade
	fadeStartTime     time.Time
	fadeStopTime      time.Time
}

func (l *guiLight) SetErr(state bool) {
	if l.err != state {
		l.err = state
	}
}

func (l *guiLight) In(x, y float32) bool {
	a := float64(l.x - x)
	b := float64(l.y - y)
	return math.Sqrt(a*a+b*b) < float64(l.size/2)
}

func (l *guiLight) MoveBy(x, y float32) {
	l.x += x
	l.y += y
	for i := 0; i < len(l.onVertices); i++ {
		inner := l.onVertices[i]
		for j := 0; j < len(inner); j++ {
			inner[j].DstX += x
			inner[j].DstY += y
		}
		inner = l.offVertices[i]
		for j := 0; j < len(inner); j++ {
			inner[j].DstX += x
			inner[j].DstY += y
		}
		inner = l.redVertices[i]
		for j := 0; j < len(inner); j++ {
			inner[j].DstX += x
			inner[j].DstY += y
		}
	}
}

func blub(centerX, centerY, size float32, colr color.RGBA) (vs [][]ebiten.Vertex, is [][]uint16) {
	halfSize := size / 2
	width1 := halfSize * .26
	width2 := halfSize * .12
	width3 := halfSize * .1
	width4 := width3 - 1

	fourthSize := size / 4
	height0 := centerY - fourthSize
	height1 := centerY - fourthSize + halfSize*0.18
	height3 := centerY - fourthSize + halfSize*0.46
	height4 := centerY - fourthSize + halfSize*0.68
	height5 := centerY - fourthSize + halfSize*0.96
	height6 := height5 + 1

	v, i := genFourSides(
		centerX-width1, height0,
		centerX+width1, height0,
		centerX+width1, height1,
		centerX-width1, height1,
		colr,
	)
	vs = append(vs, v)
	is = append(is, i)

	v, i = genFourSides(
		centerX+width1, height1,
		centerX-width1, height1,
		centerX-width1, height3,
		centerX+width1, height3,
		colr,
	)
	vs = append(vs, v)
	is = append(is, i)

	v, i = genFourSides(
		centerX-width1, height3,
		centerX+width1, height3,
		centerX+width2, height4,
		centerX-width2, height4,
		colr,
	)
	vs = append(vs, v)
	is = append(is, i)

	v, i = genFourSides(
		centerX-width2, height4,
		centerX+width2, height4,
		centerX+width3, height4,
		centerX-width3, height4,
		colr,
	)
	vs = append(vs, v)
	is = append(is, i)

	v, i = genFourSides(
		centerX-width3, height4,
		centerX+width3, height4,
		centerX+width3, height5,
		centerX-width3, height5,
		colr,
	)
	vs = append(vs, v)
	is = append(is, i)

	v, i = genFourSides(
		centerX-width3, height5,
		centerX+width3, height5,
		centerX+width4, height6,
		centerX-width4, height6,
		colr,
	)
	vs = append(vs, v)
	is = append(is, i)

	v, i = genLine(
		centerX-width1, height1,
		centerX+width1, height1,
		black,
	)
	vs = append(vs, v)
	is = append(is, i)

	delta := halfSize * .03
	for _, x := range []float32{1, 3, 5} {
		v, i = genFourSides(
			centerX-width3, height5-delta*x,
			centerX+width3, height5-delta*x,
			centerX+width3, height5-delta*(x+1),
			centerX-width3, height5-delta*(x+1),
			black,
		)
		vs = append(vs, v)
		is = append(is, i)
	}
	return
}

func disc(centerX, centerY, outer, inner float32, colr color.RGBA) (vs [][]ebiten.Vertex, is [][]uint16) {
	vs = make([][]ebiten.Vertex, 0)
	is = make([][]uint16, 0)

	//blue disc
	// bigger blue circle
	v, i := genCircleVerticesWithColor(60, centerX, centerY, outer, colr)
	vs = append(vs, v)
	is = append(is, i)

	// inner black circle
	v, i = genCircleVerticesWithColor(60, centerX, centerY, inner, black)
	vs = append(vs, v)
	is = append(is, i)
	return
}

func multipleLights(centerX, centerY, size float32, colr color.RGBA) (vs [][]ebiten.Vertex, is [][]uint16) {

	//blue disc
	vs, is = disc(centerX, centerY, size/2, (size/2)*.94, colr)

	diffX := size / 2 * 0.28
	diffY := size / 2 * 0.05

	//draw the three blubs (there's a fourth for the shadow)
	bv, bi := blub(centerX-diffX, centerY+diffY, size*0.8, colr)
	vs = append(vs, bv...)
	is = append(is, bi...)

	bv, bi = blub(centerX+diffX, centerY+diffY, size*0.8, colr)
	vs = append(vs, bv...)
	is = append(is, bi...)

	bv, bi = blub(centerX, centerY, size*1.38, black)
	vs = append(vs, bv...)
	is = append(is, bi...)

	bv, bi = blub(centerX, centerY, size, colr)
	vs = append(vs, bv...)
	is = append(is, bi...)

	return
}
func simpleCircle(centerX, centerY, size float32, colr color.RGBA) (vs [][]ebiten.Vertex, is [][]uint16) {
	//outer blue disc
	vs, is = disc(centerX, centerY, size/2, (size/2)*.94, colr)

	//inner blue disc
	vs1, is1 := disc(centerX, centerY, size/2*.3, (size/2)*.24, colr)

	vs = append(vs, vs1...)
	is = append(is, is1...)

	width := size / 2 * .08
	x0 := centerX - width
	y0 := centerY - (size/2)*0.3
	x1 := centerX + width
	y1 := y0 + (size/2)*0.3
	v, i := genFourSides(x0, y0, x1, y0, x1, y1, x0, y1, black)
	vs = append(vs, v)
	is = append(is, i)

	width = size / 2 * .04
	x0 = centerX - width
	y0 = centerY - (size/2)*0.3
	x1 = centerX + width
	y1 = y0 + (size/2)*0.3
	v, i = genFourSides(x0, y0, x1, y0, x1, y1, x0, y1, colr)
	vs = append(vs, v)
	is = append(is, i)

	return
}

func (l *guiLight) SetOn(on bool) {
	if l.on != on {
		fade := OnFading
		if !l.on {
			fade = OffFading
		}
		l.SetFade(fade, true)
		l.on = on
	}
}

func (l *guiLight) Generate() {
	if !l.simple {
		l.onVertices, l.indices = multipleLights(l.x, l.y, l.size, blue)
		l.offVertices, _ = multipleLights(l.x, l.y, l.size, grey)
		l.redVertices, _ = multipleLights(l.x, l.y, l.size, red)
	} else {
		l.onVertices, l.indices = simpleCircle(l.x, l.y, l.size, blue)
		l.offVertices, _ = simpleCircle(l.x, l.y, l.size, grey)
		l.redVertices, _ = simpleCircle(l.x, l.y, l.size, red)
	}

}

func (l *guiLight) SetFade(state Fade, lock bool) {
	// this will set the fading
	if l.fadeState == state {
		return
	}
	if lock {
		l.drawing.Lock()
		defer l.drawing.Unlock()
	}
	l.fadeState = state

	//we need to check if the last fade was complete
	now := time.Now()
	if now.After(l.fadeStopTime) {
		//if it was done then we simply setup for the next one
		l.fadeStartTime = now

	} else {
		//however not we need to reverse direction

		left := l.fadeStopTime.Sub(now)
		l.fadeStartTime = now.Add(-left)
	}

	l.fadeStopTime = l.fadeStartTime.Add(FadingTime)
}

func (l *guiLight) Update(screen *ebiten.Image) {
	if len(l.indices) == 0 || len(l.onVertices) == 0 || len(l.offVertices) == 0 || len(l.redVertices) == 0 {
		l.Generate()
	}
	l.drawing.Lock()
	defer l.drawing.Unlock()
	options := &ebiten.DrawTrianglesOptions{}
	//next let's determine which direction and if we should still be fading
	now := time.Now()
	if now.Before(l.fadeStopTime) {

		n := float64(now.UnixNano())
		start := float64(l.fadeStartTime.UnixNano())
		stop := float64(l.fadeStopTime.UnixNano())

		lerp := (n - start) / (stop - start)
		var alpha float64
		switch l.fadeState {
		case OffAppearing:
			fallthrough
		case OnAppearing:
			alpha = lerp
		case OffFading:
			fallthrough
		case OnFading:
			alpha = 1 - lerp
		}

		options.ColorM.Scale(1, 1, 1, alpha)
		//options.ColorM.Translate(1, 1, 1, alpha)
	} else {
		if l.fadeState == OnFading || l.fadeState == OffFading {
			if l.on {
				l.SetFade(OnAppearing, false)
			} else {
				l.SetFade(OffAppearing, false)
			}
			return
		}
	}

	var vertices [][]ebiten.Vertex
	switch {
	case l.fadeState == OffAppearing || l.fadeState == OffFading:
		vertices = l.offVertices
	case l.fadeState == OnAppearing || l.fadeState == OnFading:
		vertices = l.onVertices
	case l.err:
		vertices = l.redVertices
	}

	for i, v := range vertices {
		ins := l.indices[i]
		screen.DrawTriangles(v, ins, emptyImage, options)
	}

	if !l.simple {
		//then we draw the number supplied
		f := getFont(l.size)
		num := fmt.Sprint(l.num)
		advance, _ := f.GlyphAdvance(([]rune(num))[0])

		text.Draw(screen, num, f, int(l.x)-advance.Round()/2, int(l.y), black)
	}
}

func getFont(size float32) font.Face {
	i := int(12.0 / 50.0 * size / 2.0)

	if _, has := normalFont[i]; !has {
		normalFont[i] = truetype.NewFace(tt, &truetype.Options{
			Size:    float64(i),
			DPI:     dpi,
			Hinting: font.HintingFull,
		})
	}
	return normalFont[i]
}

func genLine(x0, y0, x1, y1 float32, clr color.RGBA) ([]ebiten.Vertex, []uint16) {
	const width = 1

	theta := math.Atan2(float64(y1-y0), float64(x1-x0))
	theta += math.Pi / 2
	dx := float32(math.Cos(theta))
	dy := float32(math.Sin(theta))

	r := float32(clr.R) / 0xff
	g := float32(clr.G) / 0xff
	b := float32(clr.B) / 0xff
	a := float32(clr.A) / 0xff

	return []ebiten.Vertex{
		{
			DstX:   x0 - width*dx/2,
			DstY:   y0 - width*dy/2,
			SrcX:   1,
			SrcY:   1,
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: a,
		},
		{
			DstX:   x0 + width*dx/2,
			DstY:   y0 + width*dy/2,
			SrcX:   1,
			SrcY:   1,
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: a,
		},
		{
			DstX:   x1 - width*dx/2,
			DstY:   y1 - width*dy/2,
			SrcX:   1,
			SrcY:   1,
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: a,
		},
		{
			DstX:   x1 + width*dx/2,
			DstY:   y1 + width*dy/2,
			SrcX:   1,
			SrcY:   1,
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: a,
		},
	}, []uint16{0, 1, 2, 1, 2, 3}
}

func genFourSides(x1, y1, x2, y2, x3, y3, x4, y4 float32, colr color.RGBA) (vertices []ebiten.Vertex, indices []uint16) {

	r := float32(colr.R) / 0xff
	g := float32(colr.G) / 0xff
	b := float32(colr.B) / 0xff
	a := float32(colr.A) / 0xff

	return []ebiten.Vertex{
		{
			DstX:   x1,
			DstY:   y1,
			SrcX:   1,
			SrcY:   1,
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: a,
		},
		{
			DstX:   x2,
			DstY:   y2,
			SrcX:   1,
			SrcY:   1,
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: a,
		},
		{
			DstX:   x3,
			DstY:   y3,
			SrcX:   1,
			SrcY:   1,
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: a,
		},
		{
			DstX:   x4,
			DstY:   y4,
			SrcX:   1,
			SrcY:   1,
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: a,
		},
	}, []uint16{0, 1, 2, 0, 2, 3}
}

func genCircleVerticesWithColor(num int, centerX, centerY, radius float32, colr color.RGBA) (vertices []ebiten.Vertex, indices []uint16) {
	vertices = []ebiten.Vertex{}
	indices = []uint16{}
	for i := 0; i < num; i++ {
		delta := float64(i) / float64(num)
		vertices = append(vertices, ebiten.Vertex{
			DstX:   radius*float32(math.Cos(2*math.Pi*delta)) + centerX,
			DstY:   radius*float32(math.Sin(2*math.Pi*delta)) + centerY,
			SrcX:   0,
			SrcY:   0,
			ColorR: float32(colr.R) / 255,
			ColorG: float32(colr.G) / 255,
			ColorB: float32(colr.B) / 255,
			ColorA: 1.0,
		})

		indices = append(indices, uint16(i), uint16(i+1)%uint16(num), uint16(num))
	}

	vertices = append(vertices, ebiten.Vertex{
		DstX:   centerX,
		DstY:   centerY,
		SrcX:   0,
		SrcY:   0,
		ColorR: float32(colr.R) / 255,
		ColorG: float32(colr.G) / 255,
		ColorB: float32(colr.B) / 255,
		ColorA: 1,
	})

	return vertices, indices
}
