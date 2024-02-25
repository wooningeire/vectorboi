package dot

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/jakecoffman/cp"
	"golang.org/x/image/colornames"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"runtime"
	"sort"
	"sync"
	"vectorboi/app/utils"
)

type Eval func(dot *Dot, population *Population) float64

func CompoundFitness(dot *Dot, pop *Population) float64 {
	base := math.Pow(dot.body.Position().Distance(pop.Scenario.Target.Center()), 4)
	switch dot.Status {
	case Scored:
		base -= 1000 // inject drugs directly into the dot
	case Dead:
		base += 1000 // punish death
	}
	return base + float64(len(dot.Kicks))*5
}

type Population struct {
	sync.Mutex
	Scenario   Scenario
	Dots       []*Dot
	KickIndex  int
	Generation int
	Time       float64
	TotalTime  float64

	Paused bool `json:"-"`

	exr          *utils.ExRand
	space        *cp.Space
	targetCenter cp.Vector
	oldWallLen 	 int
	fitness      Eval
	best         struct {
		dot     *Dot
		fitness float64
	}
}

func (p *Population) Len() int           { return len(p.Dots) }
func (p *Population) Less(i, j int) bool { return p.Dots[i].fitness <= p.Dots[j].fitness }
func (p *Population) Swap(i, j int)      { p.Dots[i], p.Dots[j] = p.Dots[j], p.Dots[i] }

func NewPopulation(scenario Scenario) *Population {
	if !scenario.Valid() {
		panic("scenario is invalid")
	}

	// create population object
	p := &Population{
		Scenario: scenario,
		Dots:     make([]*Dot, scenario.Size, scenario.Size),
		exr:      utils.NewExRand(scenario.Seed),
		space:    cp.NewSpace(),
		fitness:  CompoundFitness,
	}

	// setup space
	p.space.SleepTimeThreshold = cp.INFINITY
	p.space.UseSpatialHash(2, p.Scenario.Size)
	p.space.SetDamping(p.Scenario.Damping)

	// 1 == dot, 2 == killwall
	ch := p.space.NewCollisionHandler(1, 2)
	ch.BeginFunc = func(arb *cp.Arbiter, space *cp.Space, userData interface{}) bool {
		a, _ := arb.Bodies()
		a.UserData.(*Dot).Inflict(Dead)
		return false
	}

	// fill population with random dots
	for i := 0; i < p.Scenario.Size; i++ {
		p.Dots[i] = randomDot(p)
	}

	p.reset()
	return p
}

func (p *Population) reset() {
	p.Generation++
	p.TotalTime += p.Time
	p.Time = 0
	p.KickIndex = 0
	for _, dot := range p.Dots {
		dot.body.SetAngle(0)
		dot.body.SetTorque(0)
		dot.body.SetAngularVelocity(0)
		dot.body.SetPosition(p.Scenario.Spawn)
		dot.body.SetVelocity(0, 0)
		dot.body.SetForce(cp.Vector{})
		dot.Inflict(Vibing)
		dot.history = nil
	}
}

type statistics struct {
	avgFitness float64
	avgMoves   float64
	avgAge     float64

	dead   int
	scored int
	vibing int
}

func (p *Population) stats() statistics {
	l := float64(len(p.Dots))
	avs := statistics{}
	for _, dot := range p.Dots {
		avs.avgFitness += dot.fitness
		avs.avgMoves += float64(len(dot.Kicks))
		avs.avgAge += float64(dot.Age)

		switch dot.Status {
		case Dead:
			avs.dead++
		case Scored:
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
	fmt.Println("Avg. Kicks:", stats.avgMoves)
	fmt.Println("Avg. Age:", stats.avgAge)
	fmt.Printf("dead %v | scored %v | vibing %v\n",
		stats.dead, stats.scored, stats.vibing)
	fmt.Println()

	// sort by fitness
	sort.Sort(p)

	for i := 0; i < (l/2)-1; i += 2 {
		j := i + l/2
		// crossover two adjacent parents
		a, b := p.crossover(p.Dots[i], p.Dots[i+1])
		a.CreatePhysicsBody(p.space)
		b.CreatePhysicsBody(p.space)

		// mutate the resulting children
		p.mutate(a)
		p.mutate(b)

		// increase the parent's Age (they survived the generation!)
		p.Dots[i].Age++
		p.Dots[i+1].Age++

		// overwrite and kill the corresponding lower half
		for _, body := range []*cp.Body{p.Dots[j].body, p.Dots[j+1].body} {
			body.EachShape(func(shape *cp.Shape) {
				p.space.RemoveShape(shape)
			})
			p.space.RemoveBody(body)
		}
		p.Dots[j] = a
		p.Dots[j+1] = b

		// todo have genetics be more random?
	}

	runtime.GC()
	p.reset()
}

func (p *Population) resetBest() {
	p.best.dot = nil
	p.best.fitness = math.Inf(1)
}

func (p *Population) Step(dt float64) {
	if p.Paused || dt == 0 {
		return
	}
	
	if len(p.Scenario.Walls) != p.oldWallLen {
		for _, wall := range p.Scenario.Walls {
			wall.PhysicsShape(p.space)
		}
		p.oldWallLen = len(p.Scenario.Walls)
	}

	if p.Time >= p.Scenario.GenerationTime {
		p.evolve()
	}

	p.space.Step(dt)

	p.Time += dt
	hitnow := p.Time >= float64(p.KickIndex)*p.Scenario.KickTime

	// find most fit dot ( O(n) lol )
	p.resetBest()
	for _, dot := range p.Dots {
		if dot.Status.Static() {
			continue
		}

		if f := p.fitness(dot, p); f < p.best.fitness {
			p.best.dot = dot
			p.best.fitness = f
		}
	}

	for _, dot := range p.Dots {
		// hit dot when the thing yea
		if hitnow {
			dot.Kick(p.KickIndex)
		}

		// kill dot if hit wall
		pos := dot.body.Position()
		if pos.X > float64(p.Scenario.Width)-2 || pos.X < 2 ||
			pos.Y > float64(p.Scenario.Height)-2 || pos.Y < 2 {
			dot.Inflict(Dead)
		}

		// score the dot if hit target
		if p.Scenario.Target.ContainsVect(pos) {
			dot.Inflict(Scored)
		}
	}

	if hitnow {
		p.KickIndex++
	}
}

//func (p *Population) FindDot(pos cp.Vector) *Dot {
//	pqi := p.space.PointQueryNearest(pos, 2, cp.ShapeFilter{Group: 1})
//}

func (p *Population) Draw(dst *ebiten.Image) {
	t := p.Scenario.Target
	ebitenutil.DrawRect(dst,
		t.L, t.B, t.R-t.L, t.T-t.B, colornames.Green)

	for _, dot := range p.Dots {
		pos := dot.body.Position()

		var c color.Color
		switch dot.Status {
		case Dead:
			c = colornames.Red
		case Scored:
			c = colornames.Gold
		default:
			c = colornames.White
		}
		if dot == p.best.dot {
			dot.DrawHistory(dst, colornames.Lightblue)
		}
		dst.Set(int(pos.X), int(pos.Y), c)
	}

	for _, wall := range p.Scenario.Walls {
		wall.Draw(dst)
	}
}
