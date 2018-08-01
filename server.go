package pixelflood_server

import (
	"net"
	//"log"
	"strconv"
	"strings"
	"fmt"
	"bufio"
	"github.com/tjclement/framebuffer"
	"math"
)

type Pixel struct {
	R uint8
	G uint8
	B uint8
}

type PixelServer struct {
	Pixels       [][]Pixel
	screenWidth  uint16
	screenHeight uint16
	framebuffer  *framebuffer.Framebuffer
	shouldRender bool
	socket       *net.Listener
	connections  []net.Conn
	udpConn		 *net.UDPConn
	shouldClose  bool
	intDict      map[string]int
	byteDict     map[string]uint8
}

func NewServer(framebuffer *framebuffer.Framebuffer, shouldRender bool, width uint16, height uint16, useUdp bool) (*PixelServer) {
	pixels := make([][]Pixel, width)
	for i := uint16(0); i < width; i++ {
		pixels[i] = make([]Pixel, height)
	}

	socket, err := net.Listen("tcp", ":1234")

	if err != nil {
		panic(err)
	}

	var udpConn *net.UDPConn

	if useUdp {
		udpAddr, _ := net.ResolveUDPAddr("udp", ":1235")
		udpConn, err = net.ListenUDP("udp", udpAddr)

		if err != nil {
			panic(err)
		}
	}

	server := PixelServer{pixels, width, height, framebuffer, shouldRender, &socket, make([]net.Conn, 0), udpConn, false, map[string]int{}, map[string]uint8{}}

	for i := 0; i < 256; i++ {
		stringVal := fmt.Sprintf("%02x", i)
		server.byteDict[stringVal] = uint8(i)
		stringVal = fmt.Sprintf("%02X", i)
		server.byteDict[stringVal] = uint8(i)
	}

	max := int(math.Max(float64(width), float64(height)))
	for i := 0; i < max; i++ {
		server.intDict[strconv.Itoa(i)] = i
	}

	return &server
}

func (server *PixelServer) Run() {
	for !server.shouldClose {
		conn, err := (*server.socket).Accept()

		if err != nil {
			//log.Println("Error accepting new connection: ", err)
			continue
		}

		server.connections = append(server.connections, conn)
		go server.handleRequest(&conn)
	}
}

func (server *PixelServer) Stop() {
	server.shouldClose = true
	for _, conn := range server.connections {
		conn.Close()
	}
	(*server.socket).Close()
}

func (server *PixelServer) runUdp() {
	payload := make([]byte, 2048)
	for !server.shouldClose {
		amount, err := server.udpConn.Read(payload)

		if err != nil {
			fmt.Println(err)
		}

		if amount < 7 {
			fmt.Println("UDP message too short")
		}

		x, y, r, g, b := uint16(payload[0] << 8 + payload[1]), uint16(payload[2] << 8 + payload[3]), payload[4], payload[5], payload[6]
		fmt.Println("UDP", x, y, r, g, b)
		server.setPixel(x, y, r, g, b)
	}
}

func (server *PixelServer) handleRequest(conn *net.Conn) {
	scanner := bufio.NewScanner(*conn)

	for !server.shouldClose && scanner.Scan() {
		data := scanner.Text()

		// Malformed packet, does not contain recognised command
		if len(data) < 1 {
			continue
		}

		// Strip newline, and split by spaces to get command components
		commandComponents := strings.Split(data, " ")

		// For every commandComponents data, pass on its components
		if len(commandComponents) > 0 {
			x, y, r, g, b, err := server.parsePixelCommand(commandComponents)
			if err == nil {
				//fmt.Println(data, "|", x, y, r, g, b)
				server.setPixel(x, y, r, g, b)
			} else {
				//fmt.Println("Error parsing:", err, data, (*conn).RemoteAddr().String())
			}
		}
	}
	if err := scanner.Err(); err != nil {
		//fmt.Println("Error reading standard input:", err)
		return
	}
	(*conn).Close()
}

func (server *PixelServer) setPixel(x uint16, y uint16, r uint8, g uint8, b uint8) {
	if x >= server.screenWidth || y >= server.screenHeight {
		return
	}

	server.Pixels[x][y].R = r
	server.Pixels[x][y].G = g
	server.Pixels[x][y].B = b

	if server.shouldRender {
		server.framebuffer.WritePixel(int(x), int(y), r, g, b, 0)
	}
}

func (server *PixelServer) parsePixelCommand(commandPieces []string) (uint16, uint16, uint8, uint8, uint8, error) {
	if len(commandPieces) != 4 {
		return 0, 0, 0, 0, 0, fmt.Errorf("Command length mismatch")
	}

	x, y := parseUint16(commandPieces[1]), parseUint16(commandPieces[2])

	if len(commandPieces[3]) != 6 {
		return 0, 0, 0, 0, 0, fmt.Errorf("RGB length mismatch %s", commandPieces[3])
	}

	pixelValue := parseHexRGB(commandPieces[3])

	r, g, b := uint8(pixelValue&0xFF0000>>16), uint8(pixelValue&0x00FF00>>8), uint8(pixelValue&0x0000FF)

	return uint16(x), uint16(y), r, g, b, nil
}

func parseHexRGB(hex string) uint32 {
	length := len(hex)
	result := uint32(0)

	for i := length - 1; i > 0; i -= 2 {
		low := hex[i]
		high := hex[i-1]

		// ¯\_(ツ)_/¯
		high = (high & 0xf) + ((high&0x40)>>6)*9
		low = (low & 0xf) + ((low&0x40)>>6)*9

		result += uint32(((high << 4) | low)) << uint((length-( i+1 ))*4)
	}
	return result
}

func parseUint16(numeric string) uint16 {
	length := len(numeric)
	var result uint16

	for i := length - 1; i >= 0; i-- {
		char := uint16(numeric[i] - 48)
		exponent := length-( i+1 )
		for time := 0; time < exponent; time++ {
			char *= 10
		}
		result += char
	}
	return result
}
