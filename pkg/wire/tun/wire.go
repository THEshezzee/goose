package tun

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/songgao/water"
	"github.com/songgao/water/waterutil"
	"github.com/pkg/errors"

	"goose/pkg/wire"
)

var (
	logger = log.New(os.Stdout, "tunwire: ", log.LstdFlags | log.Lshortfile)
	// manager
	tunWireManager *TunWireManager
)


const (
	// max receive buffer size
	tunBuffSize = 2048
	// ignored routing
	defaultRouting = "0.0.0.0/0"
)

// register ipfs wire manager
func init() {
	tunWireManager = newTunWireManager()
	wire.RegisterWireManager(tunWireManager)
}


// tun device
type TunWire struct {
	// base
	wire.BaseWire
	// tun interface
	ifTun *water.Interface
	// address
	address net.IP
	// gateway
	gateway net.IP
	// local network
	network net.IPNet
	// provided network
	providedNetwork []net.IPNet
	// accepted network
	acceptedNetwork []net.IPNet
}

// Encode
func (w *TunWire) Encode(msg *wire.Message) error {

	switch msg.Type {

	case wire.MessageTypePacket:
		if err := w.writePacket(msg); err != nil {
			return err
		}
	case wire.MessageTypeRouting:
		if routing, ok := msg.Payload.(wire.Routing); ok {
			return w.setupHostRouting(routing.Routings)
		} else {
			return errors.Errorf("invalid routing message %+v", msg)
		}
	}
	return nil
}


// Decode
func (w *TunWire) Decode(msg *wire.Message) error {

	if err := w.readPacket(msg); err != nil {
		return err
	}
	return nil
}

func (w *TunWire) Close() error {
	return w.ifTun.Close()
}


func (w *TunWire) readPacket(msg *wire.Message) error {
	buff :=	make([]byte, tunBuffSize)
	for {
		n, err := w.ifTun.Read(buff)
		if err != nil {
			return errors.WithStack(err)
		}
		if !waterutil.IsIPv4(buff) {
			logger.Printf("recv: ignore none ipv4 packet len %d", n)
			continue
		} else {
			msg.Type = wire.MessageTypePacket
			msg.Payload = wire.Packet{
				Src: waterutil.IPv4Source(buff),
				Dst: waterutil.IPv4Destination(buff),
				Data: buff[0:n],
			}
			return nil
		}
	}
}

func (w *TunWire) writePacket(msg *wire.Message) error {
	packete, ok := msg.Payload.(wire.Packet)
	if !ok {
		return errors.Errorf("got invalid packet struct %+v", msg.Payload)
	}
	if !waterutil.IsIPv4(packete.Data) {
		logger.Printf("sent: not ipv4 packet len %d", len(packete.Data))
		return nil
	}
	if _, err := w.ifTun.Write(packete.Data); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (w *TunWire) setupHostRouting(routings []net.IPNet) error {
	// route host traffic to this tun interface

	add := []net.IPNet{}
	remove := []net.IPNet{}
	for _, network := range routings {
		// ignore defult routing
		netString := network.String()
		if netString == defaultRouting {
			continue
		}
		// TODO: ignore already contained network
		add = append(add, network)
	}
	if err := setRouting(add, remove, w.gateway.String()); err != nil {
		return err
	}
	return nil
}

// Tun-wire manager
type TunWireManager struct {
	wire.BaseWireManager
}

func newTunWireManager() *TunWireManager {
	return &TunWireManager{
		BaseWireManager: wire.NewBaseWireManager(),
	}
}

func (m *TunWireManager) Dial(endpoint string) error {

	seg := strings.Split(endpoint, "/")
	if len(seg) != 3 {
		return errors.Errorf("invalid tun endpoint %s", endpoint)
	}
	name := seg[0]
	address := fmt.Sprintf("%s/%s", seg[1], seg[2])
	
	w, err := NewTunWire(name, address)
	if err != nil {
		return err
	}
	m.Out <- w
	return nil
}

func (m *TunWireManager) Protocol() string {
	return "tun"
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}


// gen a default gateway from cidr address
func defaultGateway(cidr string) (net.IP, error) {
	var gateway net.IP 	
	address, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return gateway, errors.WithStack(err)
	}
	gateway = network.IP
	inc(gateway)
	if gateway.Equal(address) {
		return gateway, errors.Errorf("%s is reserved for gateway", address.String())
	}
	return gateway, nil
}
