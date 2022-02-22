package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/jakecoffman/cp"
	"golang.org/x/image/colornames"
	"vectorboi/helpers"
)

type vec = cp.Vector
const (
	TimeStep = 1. / 60
)

var PPM = float64(10)


// ball properties
var (
	radius = 1.828 / 2
	mass = 83.9146
	moment         = cp.MomentForCircle(mass, 0, radius, vec{})
	initialBallPos = vec{20, 5}
	initialBallVel = vec{-2, -1}
)

// ground properties
var (
	groundA = vec{X: 0, Y: 30}
	groundB = vec{X: 40, Y: 40}
)


var (
	red = colornames.Red
	orange = colornames.Orange
)

func pixelize(v vec) vec {
	return v.Mult(PPM)
}

type MulticircleBody struct {
	*cp.Body
	circles []*cp.Circle
}

type MultishapeGame struct {
	space *cp.Space
	multi *cp.Body
	ground *cp.Shape

	off vec
	last vec

	paused bool
}

func (p *MultishapeGame) Init()  {
	p.space = cp.NewSpace()
	p.space.SetGravity(vec{X: 0, Y: 9.8})
	p.space.Iterations = 10

	ground := cp.NewSegment(p.space.StaticBody, groundA, groundB, 0)
	ground.SetFriction(1)
	p.ground = p.space.AddShape(ground)
	//p.ground.SetElasticity(0.9)

	p.multi = cp.NewBody(mass, moment)
	p.multi.SetPosition(initialBallPos)
	p.multi.SetVelocityVector(initialBallVel)
	ballShape := p.space.AddShape(cp.NewCircle(p.multi, radius, vec{}))
	ballShape.SetFriction(0.7)
	p.space.AddBody(p.multi)
}

func (p *MultishapeGame) Shutdown() {}

func (p *MultishapeGame) Update() error {
	switch {
	//case inpututil.IsKeyJustReleased(ebiten.KeyR):
	//	circleShader = helpers.MustLoadShader("research/public/circle.vert")
	case inpututil.IsKeyJustReleased(ebiten.KeySpace):
		p.paused = !p.paused
	case inpututil.IsKeyJustReleased(ebiten.KeyF):
		p.multi.SetPosition(initialBallPos)
		p.multi.SetVelocityVector(initialBallVel)
	}

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		tx, ty := ebiten.CursorPosition()
		x, y := float64(tx), float64(ty)
		if p.last.Length() > 0 {
			p.off.X += x - p.last.X
			p.off.Y += y - p.last.Y
		}
		p.last.X, p.last.Y = x, y
	}
	p.last.Mult(0)

	_, dppm := ebiten.Wheel()
	PPM += dppm
	if PPM < 1 {
		PPM = 1
	}

	if !p.paused {
		p.space.Step(TimeStep)
	}

	return nil
}

func (p *MultishapeGame) Draw(screen *ebiten.Image) {
	ball := pixelize(p.multi.Position())
	ebitenutil.DebugPrint(screen, fmt.Sprintf("%.5v; %.5v, %.5v", ebiten.CurrentTPS(), ball.X, ball.Y))

	a := pixelize(groundA).Add(p.off)
	b := pixelize(groundB).Add(p.off)
	ebitenutil.DrawLine(screen, a.X, a.Y, b.X, b.Y, red)
	helpers.DrawCircle(screen, ball.Add(p.off), radius * PPM, orange)
	//ebitenutil.DrawRect(screen, ball.X, ball.Y, 10, 10, Red)

}

func (p *MultishapeGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func main() {
	ebiten.SetWindowResizable(true)
	helpers.RunGame(new(MultishapeGame))
}