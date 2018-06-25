package main

import (
	"github.com/tjclement/pixelflut-go"
	"fmt"
	"flag"
	"runtime/pprof"
	"os"
	"log"
	"github.com/tjclement/framebuffer"
	"os/signal"
	"net/http"
	"github.com/nareix/joy4/format/flv"
	"github.com/nareix/joy4/av/avutil"
	"sync"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/nareix/joy4/format/rtmp"
	"io"
	"github.com/nareix/joy4/format/mp4"
	"image"
	"github.com/gqf2008/codec"
	draw "image/draw"
)

func main() {
	screen_width := flag.Int("screen_width", 320, "Width of the screen to draw on")
	screen_height := flag.Int("screen_height", 400, "Height of the screen to draw on")
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

	fb, err := framebuffer.Init(*display)

	if err != nil {
		log.Panic(err)
	}
	fb.Clear(0, 0, 0, 0)

	fmt.Println("Starting server")
	server := pixelflood_server.NewServer(fb, uint16(*screen_width), uint16(*screen_height))
	go server.Run()
	defer server.Stop()

	go startRtspServer(server, *screen_width, *screen_height)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<- c
}

type writeFlusher struct {
	httpflusher http.Flusher
	io.Writer
}

func (self writeFlusher) Flush() error {
	self.httpflusher.Flush()
	return nil
}

func startRtspServer(pixelServer *pixelflood_server.PixelServer, width int, height int) {
	server := &rtmp.Server{}

	l := &sync.RWMutex{}
	type Channel struct {
		que *pubsub.Queue
	}
	channels := map[string]*Channel{}

	l.Lock()
	ch := channels["pixelflood"]
	if ch == nil {
		ch = &Channel{}
		ch.que = pubsub.NewQueue()
		channels["pixelflood"] = ch
	} else {
		ch = nil
	}
	l.Unlock()

	var nal [][]byte
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	start := canvas.Bounds().Min

	c, _ := codec.NewH264Encoder(width, height, image.YCbCrSubsampleRatio420)
	nal = append(nal, c.Header)

	for x, _ := range pixelServer.Pixels {
		for y, _ := range pixelServer.Pixels[x] {
			pixel := pixelServer.Pixels[x][y]
			canvas.Pix[y * width * 4 + x * 4] = pixel.R
			canvas.Pix[y * width * 4 + x * 4 + 1] = pixel.G
			canvas.Pix[y * width * 4 + x * 4 + 2] = pixel.B
			canvas.Pix[y * width * 4 + x * 4 + 3] = 0
		}
	}

	img := image.YCbCr{}
	//img := image.NewYCbCr(canvas.Rect, image.YCbCrSubsampleRatio420)
	draw.Draw(img, canvas.Rect, canvas, start, draw.Src)

	for i := 0; i < 60; i++ {
		img := draw.Image{}
		p, _ := c.Encode(*img)
		if len(p.Data) > 0 {
			nal = append(nal, p.Data)
		}
	}
	for {
		// flush encoder
		p, err := c.Encode(nil)
		if err != nil {
			break
		}
		nal = append(nal, p.Data)
	}

	server.HandlePlay = func(conn *rtmp.Conn) {
		l.RLock()
		ch := channels[conn.URL.Path]
		l.RUnlock()

		if ch != nil {
			cursor := ch.que.Latest()
			avutil.CopyFile(conn, cursor)
		}
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		l.RLock()
		ch := channels[r.URL.Path]
		l.RUnlock()

		if ch != nil {
			w.Header().Set("Content-Type", "video/x-flv")
			w.Header().Set("Transfer-Encoding", "chunked")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(200)
			flusher := w.(http.Flusher)
			flusher.Flush()

			muxer := flv.NewMuxerWriteFlusher(writeFlusher{httpflusher: flusher, Writer: w})
			cursor := ch.que.Latest()

			avutil.CopyFile(muxer, cursor)
		} else {
			http.NotFound(w, r)
		}
	})

	go http.ListenAndServe(":8089", nil)

	server.ListenAndServe()

	// ffmpeg -re -i movie.flv -c copy -f flv rtmp://localhost/movie
	// ffmpeg -f avfoundation -i "0:0" .... -f flv rtmp://localhost/screen
	// ffplay http://localhost:8089/movie
	// ffplay http://localhost:8089/screen
}