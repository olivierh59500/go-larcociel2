// Package demo layers DMA Intro 2's scenery, projected sprites and scrolltext.
package demo

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/olivierh59500/democonstructionkit/geometry"
	"github.com/olivierh59500/democonstructionkit/motion"
	"github.com/olivierh59500/democonstructionkit/scrolling"
	"github.com/olivierh59500/democonstructionkit/sound"
	playback "github.com/olivierh59500/democonstructionkit/sound/ebiten"
	"github.com/olivierh59500/democonstructionkit/sprites"
	"github.com/olivierh59500/go-larcociel2/assets"
	"github.com/olivierh59500/go-larcociel2/internal/source"
)

const Width, Height, FPS = 320, 200, 50

type Game struct {
	clock      *source.Clock
	images     []*ebiten.Image
	balls      *sprites.ImageSlots
	strips     [31][4]*ebiten.Image
	stream     *scrolling.SliceStream
	scroll     *scrolling.Scrolling
	body, text *ebiten.Image
	slots      [6]sprites.ImageSlot
	player     *playback.Player
	visual     *sound.Stream
	pcm        [960 * 8]byte
	shader     *ebiten.Shader
	palette    [64]float32
	raster     [20 * 4]float32
	uniforms   map[string]any
	center     [2]int16
	closed     bool
}

func resource(name string) ([]byte, error) { return assets.Files.ReadFile("original/" + name) }
func rgb(dst []float32, w uint16) {
	dst[0] = float32(w>>8&7) * 34 / 255
	dst[1] = float32(w>>4&7) * 34 / 255
	dst[2] = float32(w&7) * 34 / 255
	dst[3] = 1
}
func NewGame(mute bool) (_ *Game, err error) {
	g := &Game{center: [2]int16{160, 80}}
	defer func() {
		if err != nil {
			g.Close()
		}
	}()
	initial, err := resource("initial-motion.bin")
	if err != nil {
		return nil, err
	}
	history, err := resource("initial-history.bin")
	if err != nil {
		return nil, err
	}
	g.clock, err = source.NewClock(initial, history)
	if err != nil {
		return nil, err
	}
	load := func(name string) (*ebiten.Image, error) {
		b, e := resource(name)
		if e != nil {
			return nil, e
		}
		im, _, e := image.Decode(bytes.NewReader(b))
		if e != nil {
			return nil, e
		}
		texture := ebiten.NewImageFromImage(im)
		g.images = append(g.images, texture)
		return texture, nil
	}
	for i := 0; i < 36; i++ {
		if _, err = load(fmt.Sprintf("background-%d.png", i)); err != nil {
			return nil, err
		}
	}
	for i := 0; i < 16; i++ {
		if _, err = load(fmt.Sprintf("ball-%d.png", i)); err != nil {
			return nil, err
		}
	}
	atlas, err := load("font.png")
	if err != nil {
		return nil, err
	}
	for i := range g.strips {
		for col := range g.strips[i] {
			g.strips[i][col] = atlas.SubImage(image.Rect(i*32+col*8, 0, i*32+col*8+8, 16)).(*ebiten.Image)
		}
	}
	g.balls, err = sprites.NewImageSlots(sprites.ImageSlotsConfig{Images: g.images, MaxSlots: 6})
	if err != nil {
		return nil, err
	}
	g.body = ebiten.NewImage(Width, Height)
	g.text = ebiten.NewImage(Width, Height)
	message, err := resource("message.txt")
	if err != nil {
		return nil, err
	}
	order, err := resource("font-order.txt")
	if err != nil {
		return nil, err
	}
	tokens := make([]scrolling.SliceToken, 0, len(message))
	for _, c := range message {
		glyph := strings.IndexByte(string(order), c)
		if glyph < 0 {
			glyph = 30
		}
		tokens = append(tokens, scrolling.SliceToken{Glyph: glyph, Width: 32})
	}
	g.stream, err = scrolling.NewSliceStream(scrolling.SliceStreamConfig{Tokens: tokens, Capacity: 36, SliceWidth: 8, Repeat: true, Initial: scrolling.DNASlice{Glyph: -1}})
	if err != nil {
		return nil, err
	}
	g.scroll, err = scrolling.New(scrolling.Config{X: 16, GlyphWindow: &scrolling.GlyphWindowConfig{Count: 36, Advance: 8, Glyph: func(slot int) scrolling.Glyph {
		s := g.stream.Slices()[(g.stream.Head()+slot)%36]
		glyph := scrolling.Glyph{Advance: 8}
		if s.Glyph >= 0 {
			glyph.Image = g.strips[s.Glyph][s.Slice]
		}
		return glyph
	}}})
	if err != nil {
		return nil, err
	}
	pal, err := resource("palette.bin")
	if err != nil {
		return nil, err
	}
	for i := 0; i < 16; i++ {
		rgb(g.palette[i*4:], binary.BigEndian.Uint16(pal[i*2:]))
	}
	rast, err := resource("raster.bin")
	if err != nil {
		return nil, err
	}
	for i := 0; i < 20; i++ {
		rgb(g.raster[i*4:], binary.BigEndian.Uint16(rast[i*2:]))
	}
	g.uniforms = map[string]any{"Palette": g.palette[:], "Raster": g.raster[:]}
	g.shader, err = ebiten.NewShader([]byte(paletteShader))
	if err != nil {
		return nil, err
	}
	music, err := resource("music.ym")
	if err != nil {
		return nil, err
	}
	g.visual, err = sound.Open("music.ym", music, sound.Options{SampleRate: 48000, BlockFrames: 960, Loop: true})
	if err != nil {
		return nil, err
	}
	if !mute {
		g.player, err = playback.Open(nil, "music.ym", music, sound.Options{SampleRate: 48000, BlockFrames: 960, Loop: true})
		if err != nil {
			return nil, err
		}
		g.player.Play()
	}
	return g, nil
}
func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.center[0]--
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.center[0]++
	}
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		g.center[1]--
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		g.center[1]++
	}
	// Touch moves the same projection center as the desktop arrow controls.
	for _, id := range ebiten.AppendTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		g.center = [2]int16{int16(x), int16(y)}
	}
	g.clock.Step()
	g.stream.Step(1, nil)
	if _, err := io.ReadFull(g.visual, g.pcm[:]); err != nil {
		return err
	}
	pose := geometry.WordPose{Matrix: [9]int16{205, 0, 0, 0, 205, 0, 0, 0, 1}}
	projection := geometry.WordProjectionConfig{DepthBias: 205, Center: g.center, ZeroDepth: geometry.WordZeroDepthHide}
	for i := range g.slots {
		p := g.clock.History[i*11]
		screen, err := geometry.ProjectWordPoint(motion.WrappedPoint{X: p.X - g.center[0], Y: p.Y - g.center[1], Z: p.Z}, pose, projection)
		if err != nil {
			return err
		}
		g.slots[i] = sprites.ImageSlot{Image: 36 + min(15, max(0, int(p.Z)>>5)), X: float64(screen.X), Y: float64(screen.Y), Hidden: !screen.Visible}
	}
	g.body.Clear()
	if err := g.balls.SetSlots(g.slots[:]); err != nil {
		return err
	}
	g.balls.Draw(g.body)
	g.text.Clear()
	g.scroll.Draw(g.text)
	return g.scroll.Err()
}
func (g *Game) Draw(dst *ebiten.Image) {
	op := ebiten.DrawRectShaderOptions{Images: [4]*ebiten.Image{g.images[(g.clock.Tick-1+36)%36], g.body, g.text}, Uniforms: g.uniforms, Blend: ebiten.BlendCopy}
	dst.DrawRectShader(Width, Height, g.shader, &op)
}
func (*Game) Layout(int, int) (int, int) { return Width, Height }
func (g *Game) Tick() int                { return g.clock.Tick }
func (g *Game) Close() {
	if g == nil || g.closed {
		return
	}
	g.closed = true
	if g.player != nil {
		g.player.Close()
	}
	if g.visual != nil {
		g.visual.Close()
	}
	if g.scroll != nil {
		g.scroll.Close()
	}
	if g.balls != nil {
		g.balls.Close()
	}
	if g.shader != nil {
		g.shader.Deallocate()
	}
	for _, im := range append(g.images, g.body, g.text) {
		if im != nil {
			im.Deallocate()
		}
	}
}

const paletteShader = `//kage:unit pixels
package main
var Palette [16]vec4
var Raster [20]vec4
func Fragment(position vec4,source vec2,color vec4)vec4{
 p:=source-imageSrc0Origin();i:=int(clamp(floor(imageSrc0At(source).r*15+.5),0,15));ball:=imageSrc1At(source)
 if ball.a>0 {i=(i/4)*4+int(floor(ball.r*15+.5))}
 if imageSrc2At(source).a>0 {i=i%8+8}
 if i==8&&p.y<16{return Raster[int(p.y)/2]*color};if i==4&&p.y>=49&&p.y<93{return Raster[9+int((p.y-49)/4)]*color};return Palette[i]*color
}
`
