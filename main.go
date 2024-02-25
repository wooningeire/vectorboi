package main

import (
	"flag"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"os"
	"vectorboi/app"
	"vectorboi/app/dot"
	"vectorboi/app/utils"
	"vectorboi/helpers"
)

const TimeStep = 1 / 10.

var RequiredFolders = [...]string{
	"snapshots",
	"scenarios",
}

var (
	startInEditor = flag.Bool("edit", false, "start in editor")
	scenarioLoc = flag.String("s", "", "import custom scenario")
)

func turbo(pop *dot.Population, control chan bool) {
	active := false

	for {
		if active {
			select {
			case active = <- control:
			default:
				pop.Step(1 / 30.)
			}
		} else {
			active = <- control
		}
	}
}

type SimulationGame struct {
	pop   *dot.Population
	editor *app.Editor
	editing bool

	turbo bool
	turboControl chan bool
}

func (s *SimulationGame) Init() {
	for _, folder := range RequiredFolders {
		if _, err := os.Stat(folder); os.IsNotExist(err) {
			os.Mkdir(folder, 777)
		}
	}
	flag.Parse()

	myscenario := dot.DefaultScenario
	if *scenarioLoc != "" {
		utils.Import(*scenarioLoc, &myscenario)
	}

	s.editing = *startInEditor
	s.pop = dot.NewPopulation(myscenario)
	s.editor = app.NewEditor(&s.pop.Scenario)
	s.turboControl = make(chan bool)
	ebiten.SetWindowSize(s.pop.Scenario.Width, s.pop.Scenario.Height)

	go turbo(s.pop, s.turboControl)

	//start := time.Now()
	//for time.Since(start) < 15 * time.Second {
	//	s.pop.Step(1. / 30)
	//}
}

func (s *SimulationGame) Shutdown() {}

func (s *SimulationGame) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		s.editing = !s.editing
	}

	if s.editing {
		s.editor.Interact()
	} else {
		if !s.turbo {
			s.pop.Step(TimeStep)
		}

		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeySpace):
			s.pop.Paused = !s.pop.Paused
		case inpututil.IsKeyJustPressed(ebiten.KeyS) && ebiten.IsKeyPressed(ebiten.KeyControl):
			utils.Export("snapshot", s.pop)
		case inpututil.IsKeyJustPressed(ebiten.KeyT):
			s.turbo = !s.turbo
			s.turboControl <- s.turbo
		}
	}

	return nil
}

func (s *SimulationGame) Draw(screen *ebiten.Image) {
	if !s.editing {
		s.pop.Draw(screen)
		msg := fmt.Sprintf("kick %v, gen %v, dt %.2f", s.pop.KickIndex, s.pop.Generation, s.pop.Time)
		if s.turbo {
			msg += "\n>>"
		}
		ebitenutil.DebugPrint(screen, msg)
	} else {
		x, y := ebiten.CursorPosition()
		msg := fmt.Sprintf("(%d, %d)", x, y)
		s.editor.Draw(screen)
		ebitenutil.DebugPrint(screen, msg)
	}
}

func (s *SimulationGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func main() {
	helpers.RunGame(new(SimulationGame))
}
