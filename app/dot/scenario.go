package dot

import (
	"github.com/jakecoffman/cp"
	"vectorboi/app/structures"
)

const (
	defw = 640
	defh = 480
)

type Chances struct {
	ChangeKick float64
	RemoveKick float64
	AddKick    float64
	SwapKick   float64
}

type Scenario struct {
	Name           string
	Seed           int
	Width, Height  int
	Size           int
	KickTime       float64
	Chances        Chances
	GenerationTime float64
	Damping        float64
	Spawn          cp.Vector
	Target         cp.BB
	Walls          []structures.KillWall
}

func (s Scenario) Valid() bool {
	switch {
	case s.Size%2 != 0:
	case s.Walls == nil:
	case s.KickTime < 0:
	case s.GenerationTime < 0:
	case s.Damping <= 0, s.Damping > 1:
	case s.Width < 0, s.Height < 0:
		return false
	}
	return true
}

// hi-yah!
var DefaultChances = Chances{
	ChangeKick: 0.1,
	RemoveKick: 0.05,
	AddKick:    0.05,
	SwapKick:   0,
}

var DefaultScenario = Scenario{
	Name:           "default",
	Seed:           42,
	Width:          defw,
	Height:         defh,
	Size:           1000,
	KickTime:       1.2,
	GenerationTime: 40,
	Damping:        0.5,
	Chances:        DefaultChances,
	Spawn: cp.Vector{
		X: defw * 0.1,
		Y: defh * 0.1,
	},
	Target: cp.NewBBForExtents(cp.Vector{
		X: defw * 0.9,
		Y: defh * 0.1,
	}, 10, 10),
	// Walls: make([]structures.KillWall, 0),
	Walls: []structures.KillWall{
		structures.MakeKillWall(
			defw * 0.2,
			0,
			defw * 0.2,
			defh * 0.5,
		),
		structures.MakeKillWall(
			defw * 0.5,
			defh,
			defw * 0.5,
			defh * 0.5,
		),
		structures.MakeKillWall(
			defw * 0.8,
			0,
			defw * 0.8,
			defh * 0.5,
		),
	},
}
