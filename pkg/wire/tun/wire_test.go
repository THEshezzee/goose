package tun


import (
	"net"
	"testing"
	"context"
	"sync"
	"time"
	"goose/pkg/wire"
	"goose/pkg/utils"
)

// test dial ipfs wire
func TestConnect(t *testing.T) {

	wires := []wire.Wire{}
	var wg sync.WaitGroup
	wg.Add(1)
	// outbount channel reader
	go func () {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
		defer cancel()
		for {
			select {
			case w, _ := <- wire.Out():
				defer w.Close()
				t.Logf("outbound wire %+v", w)
				wires = append(wires, w)
			case <- ctx.Done():
				t.Log("wait for wire timed out")
				return
			}
		}	
	} ()
	if err := wire.Dial("tun", "goose1/192.168.100.2/24"); err != nil {
		t.Logf("%+v", err)
		t.Fail()
	}
	if err := wire.Dial("tun", "goose2/192.168.101.2/24"); err != nil {
		t.Logf("%+v", err)
		t.Fail()
	}
	wg.Wait()
	if len(wires) != 2 {
		t.Logf("wire count not matched %+v", wires)
		t.Fail()
	}
}

// test wire read
func TestTraffic(t *testing.T) {

	ping := make(chan wire.Packet)
	// outbount channel reader
	go func () {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
		defer cancel()
		for {
			select {
			case w, _ := <- wire.Out():
				defer w.Close()
				t.Logf("outbound wire %+v", w)
				// send routing messages to wire
				_, ipnet1, _ := net.ParseCIDR("10.1.0.0/16")
				_, ipnet2, _ := net.ParseCIDR("10.2.2.0/24")
				msg := wire.Message{
					Type: wire.MessageTypeRouting,
					Payload: wire.Routing{
						Routings: []net.IPNet{*ipnet1, *ipnet2},
					},
				}
				if err := w.Encode(&msg); err != nil {
					t.Logf("%+v", err)
					t.Fail()
				}
				// expect to find the ping message
				dst, _, _ := net.ParseCIDR("10.2.2.1/32")
				for {
					if err := w.Decode(&msg); err != nil {
						t.Logf("%+v", err)
						t.Fail()
					}
					t.Logf("got one packet %+v", msg)
					if msg.Type == wire.MessageTypePacket {
						packet := msg.Payload.(wire.Packet)
						if packet.Dst.Equal(dst) {
							ping <- packet
						}
					}
				}
				return
			case <- ctx.Done():
				t.Log("wait for wire timed out")
				t.Fail()
				return
			}
		}	
	} ()
	// dial wire
	if err := wire.Dial("tun", "goose1/192.168.100.3/24"); err != nil {
		t.Logf("%+v", err)
		t.Fail()
	}
	// ping the wire.
	// fake ip, so ignore none zero return code
	go func () {
		if out, err := utils.RunCmd("ping", "-c", "30", "10.2.2.1"); err != nil {
			t.Logf("%+v %s", err, out)
		}
	} ()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 30)
	defer cancel()
	select {
	case <- ping:
		return
	case <- ctx.Done():
		t.Log("not any ping messages")
		t.Fail()
	}
}