package main

import (
	"github.com/tjclement/pixelflood_server"
	"fmt"
	"flag"
	"runtime/pprof"
	"os"
	"log"
)

func main() {
	screen_width := flag.Int("screen_width", 320, "Width of the screen to draw on")
	screen_height := flag.Int("screen_height", 320, "Height of the screen to draw on")
	display := flag.String("display", "/dev/fb0", "Name of the framebuffer device to write to")
	profile := flag.Bool("profile", false, "Set to true to enable CPU profiling > cpu.profile")
	flag.Parse()

	if *profile {
		fmt.Println("Running with CPU profiling enabled")
		prof_file, err := os.Create("cpu.profile")
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(prof_file)
		defer pprof.StopCPUProfile()
	}

	fmt.Println("Starting server")
	server := pixelflood_server.NewServer(uint16(*screen_width), uint16(*screen_height))
	go server.Run()
	defer server.Stop()


	fmt.Println("Starting render thread")
	renderer := pixelflood_server.NewRenderer(server, *display, uint16(*screen_width), uint16(*screen_height))
	renderer.Initialise()
	go renderer.Run()
	defer renderer.Stop()

	<- make(chan int)
}
