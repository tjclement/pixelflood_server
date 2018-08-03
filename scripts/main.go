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
	screen_width := flag.Int("screen_width", 620, "Width of the screen to draw on")
	screen_height := flag.Int("screen_height", 200, "Height of the screen to draw on")
	display := flag.String("display", "/dev/fb0", "Name of the framebuffer device to write to")
	profile := flag.Bool("profile", false, "Set to true to enable CPU profiling > cpu.profile")
	proxy := flag.Bool("proxy", false, "Set to true to disable writing to screen, and instead proxy to other pixelflood servers")
	udp := flag.Bool("udp", false, "Set to true to use UDP on port 1235 with the binary protocol")
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
		fmt.Println("Initialising framebuffer mapping")
		fb_init, err := framebuffer.Init(*display)
		fb = fb_init

		if err != nil {
			fmt.Printf("Error setting framebuffer: %s\r\n", err.Error())
		}
		fb.Clear(0, 0, 0, 0)
	}

	fmt.Println("Starting server")
	server := pixelflood_server.NewServer(fb, !*proxy, uint16(*screen_width), uint16(*screen_height), *udp)
	go server.Run()
	defer server.Stop()

	if *proxy {
		fmt.Println("Starting proxies")
		proxy1 := pixelflood_server.NewProxy("pixelpush1.campzone.lan:1235", 0, 0, uint16(*screen_width/2), uint16(*screen_height), server)
		fmt.Println("Connecting proxy 1")
		err := proxy1.Connect()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Successfully started proxy 1")
		go proxy1.Run()
		defer proxy1.Stop()

		//proxy2 := pixelflood_server.NewProxy("pixelpush1.campzone.lan:1234", uint16(*screen_width/2), 0, uint16(*screen_width), uint16(*screen_height), server)
		//fmt.Println("Connecting proxy 2")
		//err = proxy2.Connect()
		//if err != nil {
		//	log.Fatal(err)
		//}
		//
		//fmt.Println("Successfully started proxy 2")
		//go proxy2.Run()
		//defer proxy2.Stop()
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}
