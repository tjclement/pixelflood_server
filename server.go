package pixelflood_server

import (
	"net"
	//"log"
	"strconv"
	"strings"
	"fmt"
	"bufio"
	"time"
	"github.com/orcaman/concurrent-map"
	"github.com/tjclement/framebuffer"
)

type Pixel struct {
	R uint8
	G uint8
	B uint8
}

type PixelServer struct {
	Pixels            [][]Pixel
	screenWidth       uint16
	screenHeight      uint16
	framebuffer		  *framebuffer.Framebuffer
	socket            *net.Listener
	clientConnections cmap.ConcurrentMap
	shouldClose       bool
}

func NewServer(framebuffer *framebuffer.Framebuffer, width uint16, height uint16) (*PixelServer) {
	pixels := make([][]Pixel, width)
	for i := uint16(0); i < width; i++ {
		pixels[i] = make([]Pixel, height)
	}

	socket, err := net.Listen("tcp", ":1234")

	if err != nil {
		panic(err)
	}

	return &PixelServer{pixels, width, height, framebuffer, &socket, cmap.New(), false}
}

func (server *PixelServer) Run() {
	for !server.shouldClose {
		//fmt.Println("Accepting")
		conn, err := (*server.socket).Accept()

		if err != nil {
			//log.Println("Error accepting new connection: ", err)
			continue
		}

		//fmt.Println("Accepted")

		go func(conn *net.Conn){
			//fmt.Println("Registering")
			ip, port := getRemoteIP(conn)
			//fmt.Println("Getting info")
			connPool, exists := server.clientConnections.Get(ip)

			//fmt.Println("Exists", exists)
			if !exists {
				//fmt.Println("Adding IP", ip)
				server.clientConnections.Set(ip, cmap.New())
				connPool, _ = server.clientConnections.Get(ip)
				connPool.(cmap.ConcurrentMap).Set(port, conn)
				go server.handleClientConnections(connPool.(cmap.ConcurrentMap))
			} else {
				//fmt.Println("IP already present", ip)
				connPool, _ = server.clientConnections.Get(ip)
				connPool.(cmap.ConcurrentMap).Set(port, conn)
			}
		}(&conn)
	}
}

func (server *PixelServer) Stop() {
	//fmt.Println("Stopping")
	server.shouldClose = true
	for client := range server.clientConnections.IterBuffered() {
		connections := client.Val.(cmap.ConcurrentMap)
		for connection := range connections.IterBuffered() {
			conn := connection.Val.(*net.Conn)
			(*conn).(net.Conn).Close()
			connections.Remove(connection.Key)
		}
	}
	(*server.socket).Close()
}

func (server *PixelServer) handleClientConnections(connections cmap.ConcurrentMap) {
	//fmt.Println("Handling")
	scanners := map[string]*bufio.Scanner{}

	for !server.shouldClose {
		for item := range connections.IterBuffered() {
			conn := item.Val.(*net.Conn)
			address := (*conn).RemoteAddr().String()
			scanner, exists := scanners[address]
			if !exists {
				scanners[address] = bufio.NewScanner(*conn)
				scanner = scanners[address]
			}

			//fmt.Println("Scanning hcc", address)
			(*conn).SetReadDeadline(time.Now().Add(1 * time.Second))
			if scanner.Scan() {
				//fmt.Println("Scanned hcc", address)
				data := scanner.Text()
				//fmt.Println("Data hcc", address)

				// Malformed packet, does not contain recognised command
				if len(data) < 1 {
					//fmt.Println("Malformed")
					continue
				}

				// Strip newline, and split by spaces to get command components
				commandComponents := strings.Split(data, " ")

				// For every commandComponents data, pass on its components
				if len(commandComponents) > 0 {
					x, y, pixel, err := parsePixelCommand(commandComponents)
					if err == nil {
						//fmt.Println("Setting")
						server.setPixel(x, y, pixel)
					}
				}
			} else if err := scanner.Err(); err != nil {
				fmt.Println("Error reading standard input:", err)
				(*conn).Close()
				connections.Remove(item.Key)
				delete(scanners, address)
			} else {
				//fmt.Println("No data")
			}
		}
	}
}

func (server *PixelServer) setPixel(x uint16, y uint16, pixel *Pixel) {
	if x >= server.screenWidth || y >= server.screenHeight {
		return
	}

	//server.Pixels[x][y] = *pixel
	server.framebuffer.WritePixel(int(x), int(y), pixel.R, pixel.G, pixel.B, 0)
}

func getRemoteIP(conn *net.Conn) (string, string) {
	address := (*conn).RemoteAddr().String()
	pieces := strings.Split(address, ":")
	return pieces[0], pieces[1]
}

func parsePixelCommand(commandPieces []string) (uint16, uint16, *Pixel, error) {
	if len(commandPieces) != 4 {
		return 0, 0, nil, fmt.Errorf("Command length mismatch")
	}

	x, err := strconv.ParseUint(commandPieces[1], 10, 16)
	if err != nil {
		return 0, 0, nil, err
	}

	y, err := strconv.ParseUint(commandPieces[2], 10, 16)
	if err != nil {
		return 0, 0, nil, err
	}

	pixelValue := commandPieces[3]
	if len(pixelValue) != 6 {
		return 0, 0, nil, fmt.Errorf("Pixel length mismatch")
	}

	r, err := strconv.ParseUint(pixelValue[0:2], 16, 8)
	if err != nil {
		return 0, 0, nil, err
	}

	g, err := strconv.ParseUint(pixelValue[2:4], 16, 8)
	if err != nil {
		return 0, 0, nil, err
	}

	b, err := strconv.ParseUint(pixelValue[4:6], 16, 8)
	if err != nil {
		return 0, 0, nil, err
	}

	return uint16(x), uint16(y), &Pixel{uint8(r), uint8(g), uint8(b)}, nil
}
