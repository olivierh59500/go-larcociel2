// Package source preserves the intro's accelerating trail motion.
package source

import (
	"encoding/binary"
	"fmt"
)

type Point struct{ X, Y, Z int16 }
type Clock struct {
	Tick, Counter                    int
	Velocity, Acceleration, Position Point
	History                          [66]Point
}

func NewClock(motion, history []byte) (*Clock, error) {
	if len(motion) != 20 || len(history) != 66*6 {
		return nil, fmt.Errorf("source: incomplete motion tables")
	}
	word := func(at int) int16 { return int16(binary.BigEndian.Uint16(motion[at:])) }
	c := &Clock{Velocity: Point{word(0), word(2), word(4)}, Acceleration: Point{word(6), word(8), word(10)}, Position: Point{word(12), word(14), word(16)}, Counter: int(word(18))}
	for i := range c.History {
		at := i * 6
		c.History[i] = Point{int16(binary.BigEndian.Uint16(history[at:])), int16(binary.BigEndian.Uint16(history[at+2:])), int16(binary.BigEndian.Uint16(history[at+4:]))}
	}
	return c, nil
}
func (c *Clock) Step() {
	copy(c.History[:65], c.History[1:])
	c.Position.X += c.Velocity.X
	c.Position.Y += c.Velocity.Y
	c.Position.Z += c.Velocity.Z
	c.History[65] = c.Position
	c.Counter++
	if c.Counter == 3 {
		c.Counter = 0
		c.Velocity.X += c.Acceleration.X
		c.Velocity.Y += c.Acceleration.Y
		c.Velocity.Z += c.Acceleration.Z
		if c.Velocity.X == 10 || c.Velocity.X == -10 {
			c.Acceleration.X = -c.Acceleration.X
		}
		if c.Velocity.Y == 7 || c.Velocity.Y == -7 {
			c.Acceleration.Y = -c.Acceleration.Y
		}
		if c.Velocity.Z == 12 || c.Velocity.Z == -12 {
			c.Acceleration.Z = -c.Acceleration.Z
		}
	}
	c.Tick++
}
