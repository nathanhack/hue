package gui

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/golang/freetype/truetype"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/examples/resources/fonts"

	"github.com/sirupsen/logrus"

	"github.com/amimof/huego"
)

var (
	normalTermination = fmt.Errorf("normal termination")
	emptyImage, _     = ebiten.NewImage(1920, 1080, ebiten.FilterDefault)
	options           = &ebiten.DrawTrianglesOptions{}
)

func init() {
	emptyImage.Fill(color.White)
	var err error
	tt, err = truetype.Parse(fonts.MPlus1pRegular_ttf)
	if err != nil {
		log.Fatal(err)
	}
}

type targetString string

type GUI struct {
	UserName         string
	GroupNum         int
	Width            int
	Height           int
	bridge           *huego.Bridge
	lights           map[int]*guiLight
	bigLight         guiLight
	screenSaver      bool
	screenSaverLight guiLight
	lastInteraction  time.Time
	updateMux        sync.RWMutex
}

func (gui *GUI) Run() error {
	ebiten.SetFullscreen(true)
	logrus.Infof("GUI running")

	gui.Width, gui.Height = ebiten.ScreenSizeInFullscreen()
	// On mobiles, ebiten.MonitorSize is not available so far.
	// Use arbitrary values.
	if gui.Width == 0 || gui.Height == 0 {
		gui.Width = 300
		gui.Height = 450
	}

	gui.getLightStates()

	logrus.Info("Ready for main loop")

	go func() {
		for {
			select {
			case <-time.After(5 * time.Second):
				// and see if when was the last time we saw data
				gui.getLightStates()
			}
		}
	}()

	s := ebiten.DeviceScaleFactor()
	if err := ebiten.Run(gui.update, int(float64(gui.Width)*s), int(float64(gui.Height)*s), 1/s, "Hue GUI Manager"); err != nil && err != normalTermination {
		return err
	}
	return nil
}

func (gui *GUI) update(screen *ebiten.Image) error {
	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		return normalTermination
	}

	if time.Since(gui.lastInteraction) > 15*time.Second {
		return gui.updateScreenSaver(screen)
	}
	return gui.updateNormalScreen(screen)
}

func (gui *GUI) updateScreenSaver(screen *ebiten.Image) error {
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		gui.lastInteraction = time.Now()
		return nil
	}

	if ebiten.IsDrawingSkipped() {
		return nil
	}

	//if not half way down move it
	if gui.screenSaverLight.y != float32(gui.Height/2) {
		gui.screenSaverLight.MoveBy(0, float32(gui.Height/2)-gui.screenSaverLight.y)
	}
	//we do side to side movements 60 seconds per cycle

	onecycle := (60 * time.Second).Nanoseconds()
	now := time.Now().UnixNano()
	x1 := float32(math.Cos(2*math.Pi/float64(onecycle)*float64(now%onecycle)))*(float32(gui.Width)/2-gui.screenSaverLight.size/2) + float32(gui.Width)/2
	x0 := gui.screenSaverLight.x
	gui.screenSaverLight.MoveBy(x1-x0, 0)
	gui.screenSaverLight.Update(screen)

	return nil
}

func (gui *GUI) updateNormalScreen(screen *ebiten.Image) error {
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		for lightId, l := range gui.lights {
			if l.In(float32(mx), float32(my)) {
				if time.Since(l.lastChange) > FadingTime*2 {
					l.lastChange = time.Now()
					newState := !l.on
					if gui.bridge != nil {
						light, err := gui.bridge.GetLight(lightId)
						if err != nil {
							logrus.Error(err)
							continue
						}

						light.SetState(huego.State{
							On: newState,
						})
					}
					l.SetOn(newState)
					gui.updateBigLight()
					gui.lastInteraction = time.Now()
				}
				return nil
			}
		}
		//at this point we should check if we've clicked the biglight
		if gui.bigLight.In(float32(mx), float32(my)) {
			if time.Since(gui.bigLight.lastChange) > FadingTime*2 {
				gui.bigLight.lastChange = time.Now()
				newState := !gui.bigLight.on
				if gui.bridge != nil {
					group, _ := gui.bridge.GetGroup(gui.GroupNum)

					group.SetState(huego.State{
						On: newState,
					})
				}
				gui.bigLight.SetOn(newState)

				for _, light := range gui.lights {
					light.SetOn(newState)
				}

				gui.lastInteraction = time.Now()
			}
			return nil
		}

	}

	if ebiten.IsDrawingSkipped() {
		return nil
	}

	gui.bigLight.Update(screen)
	for _, l := range gui.lights {
		l.Update(screen)
	}
	return nil
}

func (gui *GUI) Error() {
	go func() {
		gui.bigLight.SetErr(true)
		time.Sleep(3 * time.Second)
		gui.bigLight.SetErr(false)
	}()
}

func (gui *GUI) updateBigLight() {
	on := false
	for _, l := range gui.lights {
		on = on || l.on
	}

	gui.bigLight.SetOn(on)
}

func (gui *GUI) updateScreenSaverLight() {
	on := false
	for _, l := range gui.lights {
		on = on || l.on
	}
	gui.screenSaverLight.SetOn(on)
}

func (gui *GUI) getLightStates() {
	gui.updateMux.Lock()
	defer gui.updateMux.Unlock()

	logrus.Infof("getLightStates()")

	err := gui.activateBridge()
	if err != nil {
		logrus.Error(err)
		return
	}

	group, err := gui.bridge.GetGroup(gui.GroupNum)
	if err != nil {
		logrus.Error(err)
		return
	}

	// we do a quick check to see there something different than our internal state
	updateRequired := len(gui.lights) != len(group.Lights)

	for _, i := range group.Lights {
		id, err := strconv.Atoi(i)
		if err != nil {
			logrus.Error(err)
			continue
		}
		_, has := gui.lights[id]

		updateRequired = updateRequired || !has
	}

	if updateRequired {
		num := float64(len(group.Lights))
		logrus.Infof("first time light setup: %v", num)
		gui.lights = make(map[int]*guiLight)

		for i, lightStrID := range group.Lights {
			logrus.Infof("setting up light: %v", lightStrID)
			id, err := strconv.Atoi(lightStrID)
			if err != nil {
				logrus.Error(err)
				continue
			}
			gui.lights[id] = &guiLight{
				x:      float32(gui.Width)/2 + float32((500+100)/2*math.Cos(2*math.Pi/num*float64(i))),
				y:      float32(gui.Height)/2 + float32((500+100)/2*math.Sin(2*math.Pi/num*float64(i))),
				size:   200,
				simple: true,
			}
		}
		gui.bigLight = guiLight{
			x:    float32(gui.Width) / 2,
			y:    float32(gui.Height) / 2,
			size: 500,
			num:  len(group.Lights),
		}

		gui.screenSaverLight = guiLight{
			x:    float32(gui.Width) / 2,
			y:    float32(gui.Height) / 2,
			size: 200,
			num:  len(group.Lights),
		}
	}

	for lightID, guiLight := range gui.lights {

		light, err := gui.bridge.GetLight(lightID)
		if err != nil {
			logrus.Error(err)
			continue
		}
		guiLight.lastSeen = time.Now()
		guiLight.label = light.Name
		guiLight.SetOn(light.IsOn())
	}

	gui.updateBigLight()
	gui.updateScreenSaverLight()
}

func (gui *GUI) activateBridge() error {
	if gui.bridge == nil {
		logrus.Info("Bridge initialization")
		var err error
		gui.bridge, err = huego.Discover()
		if err != nil {
			return err
		}
		gui.bridge = gui.bridge.Login(gui.UserName)
	}

	if gui.bridge == nil {
		return fmt.Errorf("bridge activation failed")
	}

	return nil
}
