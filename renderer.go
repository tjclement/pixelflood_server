package pixelflood_server

import (
	"github.com/tjclement/framebuffer"
	"log"
	"time"
)

type Renderer struct {
	server       *PixelServer
	frameBuffer  *framebuffer.Framebuffer
	screenWidth  uint16
	screenHeight uint16
	shouldClose  bool
}

func NewRenderer(server *PixelServer, display string, width uint16, height uint16) (*Renderer) {
	fb, err := framebuffer.Init(display)

	if err != nil {
		log.Panic(err)
	}

	return &Renderer{server, fb, width, height, false}
}

func (renderer *Renderer) Initialise() {
	(*renderer.frameBuffer).Clear(0, 0, 0, 0)
}

func (renderer *Renderer) Run() {
	for !renderer.shouldClose {
		width, height := int(renderer.screenWidth), int(renderer.screenHeight)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				pixel := renderer.server.Pixels[x][y]
				renderer.frameBuffer.WritePixel(x, y, pixel.R, pixel.G, pixel.B, 0)
			}
		}
		time.Sleep(16666 * time.Microsecond) // 16.666 ms, 1 frame in 60fps refresh rate
	}
}

func (renderer *Renderer) Stop() {
	renderer.shouldClose = true
}
