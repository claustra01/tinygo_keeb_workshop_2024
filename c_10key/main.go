package main

import (
	"image/color"
	"machine"
	"machine/usb/hid/keyboard"
	"machine/usb/hid/mouse"
	"math"
	"time"

	pio "github.com/tinygo-org/pio/rp2-pio"
	"github.com/tinygo-org/pio/rp2-pio/piolib"
	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/ssd1306"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/gophers"
)

// LED definitions

type WS2812B struct {
	Pin machine.Pin
	ws  *piolib.WS2812B
}

func NewWS2812B(pin machine.Pin) *WS2812B {
	s, _ := pio.PIO0.ClaimStateMachine()
	ws, _ := piolib.NewWS2812B(s, pin)
	ws.EnableDMA(true)
	return &WS2812B{
		ws: ws,
	}
}

func (ws *WS2812B) WriteRaw(rawGRB []uint32) error {
	return ws.ws.WriteRaw(rawGRB)
}

// display definitions

type RotatedDisplay struct {
	drivers.Displayer
}

func (d *RotatedDisplay) Size() (x, y int16) {
	return y, x
}

func (d *RotatedDisplay) SetPixel(x, y int16, c color.RGBA) {
	_, sy := d.Displayer.Size()
	d.Displayer.SetPixel(y, sy-x, c)
}

func main() {

	// variables

	white := uint32(0xFFFFFFFF)
	black := uint32(0x00000000)

	colors := [][]uint32{
		{white, white, white},
		{white, white, white},
		{white, white, white},
		{white, white, white},
	}

	keyMap := [][]keyboard.Keycode{
		{keyboard.KeyBackspace, keyboard.Key0, keyboard.KeyEnter},
		{keyboard.Key1, keyboard.Key2, keyboard.Key3},
		{keyboard.Key4, keyboard.Key5, keyboard.Key6},
		{keyboard.Key7, keyboard.Key8, keyboard.Key9},
	}

	textColor := color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}

	charMap := [][]string{
		{"A", "B", "C"},
		{"D", "E", "F"},
		{"G", "I", "J"},
		{"M", "N", "P"},
	}

	// init LED

	ws := NewWS2812B(machine.GPIO1)

	colPins := []machine.Pin{
		machine.GPIO5,
		machine.GPIO6,
		machine.GPIO7,
		machine.GPIO8,
	}

	rowPins := []machine.Pin{
		machine.GPIO9,
		machine.GPIO10,
		machine.GPIO11,
	}

	for _, c := range colPins {
		c.Configure(machine.PinConfig{Mode: machine.PinOutput})
		c.Low()
	}

	for _, c := range rowPins {
		c.Configure(machine.PinConfig{Mode: machine.PinInputPulldown})
	}

	// init Keyboard

	kb := keyboard.Port()

	// init Display

	machine.I2C0.Configure(machine.I2CConfig{
		Frequency: 2.8 * machine.MHz,
		SDA:       machine.GPIO12,
		SCL:       machine.GPIO13,
	})

	display := ssd1306.NewI2C(machine.I2C0)
	display.Configure(ssd1306.Config{
		Address: 0x3C,
		Width:   128,
		Height:  64,
	})
	display.ClearDisplay()
	rotDisplay := RotatedDisplay{&display}

	tinyfont.WriteLine(&rotDisplay, &gophers.Regular58pt, 10, 70, "H", textColor)
	rotDisplay.Display()

	// init joystick

	machine.InitADC()

	ax := machine.ADC{Pin: machine.GPIO29}
	ax.Configure(machine.ADCConfig{})
	ay := machine.ADC{Pin: machine.GPIO28}
	ay.Configure(machine.ADCConfig{})

	btn := machine.GPIO0
	btn.Configure(machine.PinConfig{Mode: machine.PinInputPullup})

	// init mouse

	m := mouse.Port()

	for {
		ws.WriteRaw([]uint32{
			colors[0][0], colors[0][1], colors[0][2],
			colors[1][0], colors[1][1], colors[1][2],
			colors[2][0], colors[2][1], colors[2][2],
			colors[3][0], colors[3][1], colors[3][2],
		})

		// keyboard

		for i, c := range colPins {
			c.High()
			for j, r := range rowPins {
				if r.Get() {
					kb.Down(keyMap[i][j])
					colors[i][j] = black
					display.ClearDisplay()
					tinyfont.WriteLine(&rotDisplay, &gophers.Regular58pt, 10, 70, charMap[i][j], textColor)
					display.Display()
				} else {
					kb.Up(keyMap[i][j])
					colors[i][j] = white
				}
			}
			c.Low()
			time.Sleep(1 * time.Millisecond)
		}

		// joystick()
		x := int(ax.Get())
		y := int(ay.Get())
		dx := -1 * (y - 0x8000) / 0x800
		dy := -1 * (x - 0x8000) / 0x800
		if math.Abs(float64(dx)) < 2 {
			dx = 0
		}
		if math.Abs(float64(dy)) < 2 {
			dy = 0
		}
		m.Move(dx, dy)

		if !btn.Get() {
			m.Press(mouse.Left)
		} else {
			m.Release(mouse.Left)
		}

	}

}
