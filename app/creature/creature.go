package creature

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/jakecoffman/cp"
	"math/rand"
)

type Creature struct {
	Body []*Node
}

func NewRandomCreature(num int) *Creature {
	creature := &Creature{
		Body: []*Node{
			NewRandomNode(),
		},
	}

	for i := 0; i < num; i++ {
	check:
		for {
			node := NewRandomNode()
			node.Parent = creature.randomBodyNode()

			for _, other := range creature.Body {
				if node.Intersects(other) {
					continue check
				}
			}

			node.Parent.Attach(node)
			creature.Body = append(creature.Body, node)
			break
		}
	}

	return creature
}

func (c *Creature) root() *Node {
	return c.Body[0]
}

func (c *Creature) randomBodyNode() *Node {
	return c.Body[rand.Intn(len(c.Body))]
}

func (c *Creature) Draw(dst *ebiten.Image)  {
}

func (c *Creature) CreatePhysicsBody(space *cp.Space) *cp.Body {
	cbody := space.AddBody(cp.NewBody(0, 0))
	cbody.UserData = c

	c.root().Do(func(n *Node) {
		nshape := space.AddShape(cp.NewCircle(cbody, n.Radius, n.Position()))
		nshape.UserData = n
		nshape.SetDensity(1)
		nshape.SetFriction(0.5)
	})

	return cbody
}