# Larcociel 2 Go

A Go/Ebitengine conversion of **DMA Intro 2**, an Atari ST intro by
**Larcociel of DMA**, using Demo Construction Kit **v1.0.13**.

```sh
go run ./cmd/larcociel2
go run ./cmd/larcociel2 -mute
```

Space or Escape closes the intro. Arrow keys move the projection center. The
320 × 200 scene runs at 50 Hz, with a moving checkerboard, the original landscape
and DMA logo, a scrolling metallic font and six depth-sized spheres.

The thirty-six floor phases, sixteen sphere sizes and font alphabet come from
the native presentation data. The trail retains sixty-six position samples,
three signed word velocities, acceleration reversals and its three-frame
velocity cadence. A test compares the complete motion history, current
position and velocity with an independent 68000 checkpoint.

DCK supplies word projection, retained sprite batches, the strip-history
transport and the common Scrolling renderer. A compact palette composition
keeps the spheres' two low bitplanes separate from the landscape and floor.
The original YM6 soundtrack is embedded and replayed through DCK. Runtime
drawing performs no GPU readback.

```sh
go test ./...
go vet ./...
go run ./cmd/larcociel2 -capture captures/preview -frame 250 -frames 1
go run ./cmd/video
```

Video export creates a three-minute 50 fps H.264/AAC MP4, a PNG poster and a
JSON report in `recordings/`, at 640 × 400 pixels. DCK synchronizes graphics and
music on one simulation clock. The website uses a VP9/Opus WebM copy.

Original production: [Demozoo](https://demozoo.org/productions/79461/).
