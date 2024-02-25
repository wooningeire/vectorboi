package dot

import (
	"encoding/json"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/jakecoffman/cp"
	"image/color"
	"math"
	"vectorboi/app/utils"
)

func RandomVector(exr *utils.ExRand) cp.Vector {
	return cp.
		ForAngle(exr.Uniform(0, 2*math.Pi)).
		Mult(exr.Uniform(50, 100))
}

type DotStatus int

const (
	Vibing = DotStatus(iota)
	Dead
	Scored
)

func (ds *DotStatus) UnmarshalJSON(bytes []byte) error {
	var proxy int
	if err := json.Unmarshal(bytes, &proxy); err != nil {
		return err
	}
	*ds = DotStatus(proxy)
	return nil
}

func (ds DotStatus) MarshalJSON() ([]byte, error) {
	if b, err := json.Marshal(int(ds)); err != nil {
		return nil, err
	} else {
		return b, nil
	}
}

func (ds DotStatus) Static() bool {
	return ds == Dead || ds == Scored
}

type Dot struct {
	Kicks  []cp.Vector
	Age    int
	Status DotStatus

	history []cp.Vector
	body    *cp.Body
	fitness float64
}

func randomDot(pop *Population) *Dot {
	dot := &Dot{Kicks: make([]cp.Vector, 0)}
	for i := 0; i < pop.exr.IntRange(30, 50); i++ {
		dot.Kicks = append(dot.Kicks, RandomVector(pop.exr))
	}
	dot.history = make([]cp.Vector, 0)
	dot.CreatePhysicsBody(pop.space)
	return dot
}

func (d *Dot) Kick(move int) {
	if move >= len(d.Kicks) {
		return
	}

	d.history = append(d.history, d.body.Position())
	d.body.ApplyImpulseAtLocalPoint(d.Kicks[move], cp.Vector{})
}

func (d *Dot) CreatePhysicsBody(space *cp.Space) {
	d.body = cp.NewBody(0, 0)
	shape := cp.NewCircle(d.body, 1, cp.Vector{})
	shape.SetCollisionType(1)
	shape.SetMass(1)
	shape.SetFilter(cp.ShapeFilter{
		Group:      1,
		Categories: cp.ALL_CATEGORIES,
		Mask:       cp.ALL_CATEGORIES,
	})
	d.body.AccumulateMassFromShapes()

	space.AddShape(shape)
	space.AddBody(d.body)
	d.body.UserData = d
}

func (d *Dot) String() string {
	return fmt.Sprintf("Dot %.2f", d.fitness)
}

func (d *Dot) Inflict(status DotStatus) {
	d.Status = status
	if d.Status.Static() {
		d.body.SetType(cp.BODY_STATIC)
	} else {
		d.body.SetType(cp.BODY_DYNAMIC)
	}
}

func (d *Dot) DrawHistory(dst *ebiten.Image, c color.Color) {
	all := append(d.history, d.body.Position())
	if len(all) == 1 {
		return
	}
	for i := 0; i < len(all)-1; i++ {
		a := all[i]
		b := all[i+1]
		ebitenutil.DrawLine(dst, a.X, a.Y, b.X, b.Y, c)
	}
}
