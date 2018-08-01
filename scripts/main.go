package main

import (
	"github.com/tjclement/pixelflood_server"
	"fmt"
	"flag"
	"runtime/pprof"
	"os"
	"log"
	"github.com/tjclement/framebuffer"
	"os/signal"
)

func main() {
	screen_width := flag.Int("screen_width", 320, "Width of the screen to draw on")
	screen_height := flag.Int("screen_height", 400, "Height of the screen to draw on")
	display := flag.String("display", "/dev/fb0", "Name of the framebuffer device to write to")
	profile := flag.Bool("profile", false, "Set to true to enable CPU profiling > cpu.profile")
	proxy := flag.Bool("proxy", false, "Set to true to disable writing to screen, and instead proxy to other pixelflood servers")
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

	var fb *framebuffer.Framebuffer

	if !*proxy {
		fb, err := framebuffer.Init(*display)

		if err != nil {
			fmt.Printf("Error setting framebuffer: %s\r\n", err.Error())
		}
		fb.Clear(0, 0, 0, 0)
	}

	fmt.Println("Starting server")
	server := pixelflood_server.NewServer(fb, !*proxy, uint16(*screen_width), uint16(*screen_height))
	go server.Run()
	defer server.Stop()

	if *proxy {
		fmt.Println("Starting proxies")
		proxy1 := pixelflood_server.NewProxy("pixelpush1.campzone.lan:1234", 0, 0, uint16(*screen_width/2), uint16(*screen_height/2), server)
		proxy2 := pixelflood_server.NewProxy("pixelpush2.campzone.lan:1234", uint16(*screen_width/2), uint16(*screen_height/2), uint16(*screen_width), uint16(*screen_height), server)
		go proxy1.Run()
		go proxy2.Run()
		defer proxy1.Stop()
		defer proxy2.Stop()
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}
