package source

import (
	"encoding/json"
	"github.com/olivierh59500/go-larcociel2/assets"
	"os"
	"reflect"
	"testing"
)

func TestMotionMatchesNativeCheckpoint(t *testing.T) {
	m, e := assets.Files.ReadFile("original/initial-motion.bin")
	if e != nil {
		t.Fatal(e)
	}
	h, e := assets.Files.ReadFile("original/initial-history.bin")
	if e != nil {
		t.Fatal(e)
	}
	c, e := NewClock(m, h)
	if e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile("testdata/native-motion-49.json")
	if e != nil {
		t.Fatal(e)
	}
	var want struct {
		Steps, Counter                   int
		Velocity, Acceleration, Position [3]int16
		History                          [66][3]int16
	}
	if e = json.Unmarshal(b, &want); e != nil {
		t.Fatal(e)
	}
	for range want.Steps {
		c.Step()
	}
	point := func(p Point) [3]int16 { return [3]int16{p.X, p.Y, p.Z} }
	var history [66][3]int16
	for i, p := range c.History {
		history[i] = point(p)
	}
	if point(c.Velocity) != want.Velocity || point(c.Acceleration) != want.Acceleration || point(c.Position) != want.Position || c.Counter != want.Counter || !reflect.DeepEqual(history, want.History) {
		t.Fatalf("motion differs from the original checkpoint: velocity=%+v position=%+v counter=%d", c.Velocity, c.Position, c.Counter)
	}
}
