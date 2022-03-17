package main

import (
	_ "embed"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/jakecoffman/cp"
	"golang.org/x/image/colornames"
	"image/color"
	"math"
	"runtime"
	"sort"

	_ "image/jpeg"
	_ "image/png"
)

const (
	GenerationTime = 10
	KickTime = 1.2
	Side = 20
)

const(
	DotCategory = 1 << uint(iota)
	KillWallCategory
	GoalCategory
)

//var (
//	gooblegop, _ = ebitenutil.NewImageFromURL("https://images.thdstatic.com/productImages/bf4a1fd8-aca2-4f0f-94a6-d188cf1ba7ea/svn/black-fence-armor-deck-post-caps-pf-acn-252b-4f_600.jpg")
//	snorb, _     = ebitenutil.NewImageFromURL("https://www.pikpng.com/pngl/b/190-1905158_scary-eye-png-transparent-creepy-eyeball-png-clipart.png")
//)
//
//var gw, gh = gooblegop.Size()
//var sw, sh = snorb.Size()

type Eval func(dot *Dot, population *Population) float64

func ConstantFitness(dot *Dot, pop *Population) float64 {
	return 1
}

func RandomFitness(dot *Dot, pop *Population) float64 {
	return uniform(1, 100)
}

func DistanceFitness(dot *Dot, pop *Population) float64 {
	center := pop.Target.Center()
	constraint := pop.Spawn.Distance(center)
	return 1. - (dot.body.Position().Distance(center) / constraint)
}

func CompoundFitness(dot *Dot, pop *Population) float64 {
	base := dot.body.Position().Distance(pop.Target.Center())
	if dot.scored {
		base -= 10 // inject black tar heroin directly into the dot
	} else if dot.dead {
		base += 10 // punish death
	}
	return base + float64(len(dot.Moves)) * 5
}

type Population struct {
	Dots          []*Dot
	Space         *cp.Space
	OnMove        int
	Width, Height int
	Generation    int
	Time          float64
	Spawn   cp.Vector
	Target  cp.BB

	killwalls []KillWall
	fitness Eval
	Paused  bool

	bestDot        *Dot
	bestDotFitness float64
}

func (p *Population) Len() int           { return len(p.Dots) }
func (p *Population) Less(i, j int) bool { return p.Dots[i].fitness <= p.Dots[j].fitness }
func (p *Population) Swap(i, j int)      { p.Dots[i], p.Dots[j] = p.Dots[j], p.Dots[i] }

func NewRandomPopulation(num, width, height int, fitness Eval) *Population {
	if num % 2 != 0 {
		panic("num must be even")
	}

	if fitness == nil {
		fitness = CompoundFitness
	}

	target := cp.Vector{
		X: Width / 2,
		Y: Height / 10,
	}

	p := &Population{
		Dots:   make([]*Dot, num, num),
		Space:  cp.NewSpace(),
		Width:  width,
		Height: height,
		Spawn: cp.Vector{
			X: Width / 2,
			Y: Height - Height / 10,
		},
		Target: cp.NewBBForExtents(target, Side / 2, Side / 2),
		fitness: fitness,
		killwalls: make([]KillWall, 0),
	}

	p.Space.SleepTimeThreshold = cp.INFINITY
	p.Space.UseSpatialHash(2, 100)

	for i := 0; i < num; i++ {
		ndot := NewRandomDot()
		ndot.CreatePhysicsBody(p.Space)
		p.Dots[i] = ndot
	}

	// 1 == dot, 2 == killwall
	ch := p.Space.NewCollisionHandler(1, 2)
	ch.BeginFunc = func(arb *cp.Arbiter, space *cp.Space, userData interface{}) bool {
		a, _ := arb.Bodies()

		dot := a.UserData.(*Dot)
		//wall := b.UserData.(KillWall)

		p.kill(dot)
		return false
	}
	//p.Space.Coll

	p.reset()
	return p
}

func (p *Population) reset() {
	p.Generation++
	p.Time = 0
	p.OnMove = 0
	p.bestDot = nil
	p.bestDotFitness = math.Inf(1)
	for _, dot := range p.Dots {
		dot.body.SetAngle(0)
		dot.body.SetTorque(0)
		dot.body.SetAngularVelocity(0)
		dot.body.SetPosition(p.Spawn)
		dot.body.SetVelocity(0, 0)
		dot.body.SetForce(cp.Vector{})
		dot.scored = false
		p.unkill(dot)
	}
}

type statistics struct {
	avgFitness float64
	avgMoves float64
	avgAge float64

	dead int
	scored int
	vibing int
}

func (p *Population) stats() statistics {
	l := float64(len(p.Dots))
	avs := statistics{}
	for _, dot := range p.Dots {
		avs.avgFitness += dot.fitness
		avs.avgMoves += float64(len(dot.Moves))
		avs.avgAge += float64(dot.Age)

		switch {
		case dot.dead:
			avs.dead++
		case dot.scored:
			avs.scored++
		default:
			avs.vibing++
		}
	}
	avs.avgFitness /= l
	avs.avgMoves /= l
	avs.avgAge /= l
	return avs
}

func (p *Population) evolve() {
	l := len(p.Dots)

	// evaluate fitness
	for _, dot := range p.Dots {
		dot.fitness = p.fitness(dot, p)
	}

	// print stats
	stats := p.stats()
	fmt.Printf("==== GENERATION %v ====\n", p.Generation)
	fmt.Println("Avg. Fitness:", stats.avgFitness)
	fmt.Println("Avg. Moves:", stats.avgMoves)
	fmt.Println("Avg. Age:", stats.avgAge)
	fmt.Printf("dead %v | scored %v | vibing %v\n",
		stats.dead, stats.scored, stats.vibing)
	fmt.Println()

	// sort by fitness
	sort.Sort(p)

	// kill lower half and mutate lower half
	//middle := p.Dots[l / 2].fitness
	//for i, dot := range p.Dots {
	//	if dot.fitness < middle {
	//		p.Dots[i] = nil // RIP
	//	}
	//}
	for i := 0; i < (l / 2) - 1; i += 2 {
		j := i + l / 2
		// crossover two adjecent parents
		a, b := Crossover(p.Dots[i], p.Dots[i + 1])

		// mutate the resulting children
		Mutate(a)
		Mutate(b)

		a.CreatePhysicsBody(p.Space)
		b.CreatePhysicsBody(p.Space)

		// increase the parent's Age (they survived the generation!)
		p.Dots[i].Age++
		p.Dots[i + 1].Age++

		// overwrite and kill the corresponding lower half
		for _, body := range []*cp.Body{p.Dots[j].body, p.Dots[j + 1].body} {
			body.EachShape(func(shape *cp.Shape) {
				p.Space.RemoveShape(shape)
			})
			p.Space.RemoveBody(body)
		}
		p.Dots[j] = a
		p.Dots[j + 1] = b

	}

	//fmt.Println(p.Dots)
	runtime.GC()
	p.reset()
}

func (p *Population) unkill(dot *Dot) {
	dot.dead = false
	dot.body.SetType(cp.BODY_DYNAMIC)
}

func (p *Population) kill(dot *Dot) {
	dot.dead = true
	dot.body.SetType(cp.BODY_STATIC)
}

func (p *Population) AddKillWalls(walls ...KillWall)  {
	for _, wall := range walls {
		wall.PhysicsShape(p.Space)
		p.killwalls = append(p.killwalls, wall)
	}
}

func (p *Population) IsBest(dot *Dot) bool {
	return dot == p.bestDot
}

func (p *Population) Step(dt float64) {
	if p.Paused {
		return
	}

	if p.Time > GenerationTime { // cahnge this for
		p.evolve()
		//p.reset()
		// todo
	}

	p.Space.Step(dt)

	hitnow := false
	p.Time += dt
	if p.Time >= float64(p.OnMove) * KickTime { // change this for faster/slower kicking
		hitnow = true
	}

	for _, dot := range p.Dots {
		//if dot.dead || dot.scored {
		//	dot.body.SetType(cp.BODY_STATIC)
		//}

		// hit dot when the thing yea
		if hitnow && p.OnMove < len(dot.Moves) {
			dot.body.ApplyImpulseAtLocalPoint(dot.Moves[p.OnMove], cp.Vector{})
		}

		// kill dot if hit wall
		pos := dot.body.Position()
		if pos.X > float64(p.Width)-2 || pos.X < 2 || pos.Y > float64(p.Height)-2 || pos.Y < 2 {
			p.kill(dot)
		}

		if p.Target.ContainsVect(pos) {
			dot.SetScored()
		}

		if f := p.fitness(dot, p); f < p.bestDotFitness {
			p.bestDot = dot
			p.bestDotFitness = f
		}
	}

	if hitnow {
		p.OnMove++
	}
}

func (p *Population) Draw(dst *ebiten.Image) {
	ebitenutil.DrawRect(dst,
		p.Target.L, p.Target.B,
		p.Target.R - p.Target.L, p.Target.T - p.Target.B,
		colornames.Green)

	//op := &ebiten.DrawImageOptions{}
	for _, dot := range p.Dots {
		pos := dot.body.Position()

		var c color.Color
		switch {
		case dot.dead:
			c = colornames.Red
		case dot.scored:
			c = colornames.Gold
		case p.IsBest(dot):
			c = colornames.Hotpink
		default:
			c = colornames.White
		}
		dst.Set(int(pos.X), int(pos.Y), c)

		//op.GeoM.Reset()
		//op.GeoM.Translate(float64(-sw) / 2, float64(-sh) / 2)
		//op.GeoM.Scale(0.02, 0.02)
		//op.ColorM.Reset()
		//switch {
		//case dot.dead:
		//	op.ColorM.RotateHue(p.Time * 10)
		//case dot.scored:
		//	pulse := math.Sin(p.Time)
		//	op.GeoM.Scale(pulse, pulse)
		//}
		//op.GeoM.Translate(pos.X, pos.Y)
		//
		//dst.DrawImage(snorb, op)
	}

	for _, wall := range p.killwalls {
		wall.Draw(dst)
	}
}
