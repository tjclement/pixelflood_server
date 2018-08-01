package pixelflood_server

import (
	"net"
	"time"
	"fmt"
)

type Proxy struct {
	server *PixelServer
	conn net.Conn
	address string
	x_begin uint16
	y_begin uint16
	x_end uint16
	y_end uint16
	shouldClose bool
}

func NewProxy(address string, x_begin, y_begin, x_end, y_end uint16, server *PixelServer) (*Proxy) {
	return &Proxy{server, nil, address, x_begin, y_begin, x_end, y_end, false};
}

func (p *Proxy) Connect() error {
	conn, err := net.DialTimeout("tcp", p.address, 5*time.Second)

	if err != nil {
		fmt.Printf("Error setting up TCP connection: %s\r\n", err.Error())
		return err
	}

	p.conn = conn
	return nil
}

func (p *Proxy) Run() {
	fmt.Println("run")
	for !p.shouldClose {
		for x := p.x_begin; x < p.x_end; x ++ {
			for y := p.y_begin; y < p.y_end; y++ {
				pixel := p.server.Pixels[x][y]
				if p.conn != nil {
					_, err := p.conn.Write([]byte(fmt.Sprintf("PX %d %d %02x%02x%02x\n", x, y, pixel.R, pixel.G, pixel.B)));
					if err != nil {
						fmt.Printf("Error writing to proxy connection: %s\r\n", err.Error())
					}
				}
			}
		}
	}

	time.Sleep(10 * time.Millisecond) // 10 ms, 100x per second
}

func (p *Proxy) Stop() {
	p.shouldClose = true
}