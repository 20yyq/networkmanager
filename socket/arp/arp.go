// @@
// @ Author       : Eacher
// @ Date         : 2023-06-29 15:13:47
// @ LastEditTime : 2023-07-04 14:15:49
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /networkmanager/socket/arp/arp.go
// @@
package arp

import (
	"net"
	"time"
	"syscall"

	"encoding/binary"

	"github.com/20yyq/packet"
	"github.com/20yyq/networkmanager/socket"
)

const (
	ETH_P_ARP 	= syscall.ETH_P_ARP
	ETH_P_IP 	= syscall.ETH_P_IP
	ETH_P_IPV6 	= syscall.ETH_P_IPV6

	ETH_P_8021AD= syscall.ETH_P_8021AD
	ETH_P_8021Q = syscall.ETH_P_8021Q
)

type ArpConn struct {
	*socket.Socket

	ifindex 	int
	proto 		uint16
	addr 		net.IP
	hardware 	net.HardwareAddr
}

func NewArpConn(name string, ifi *net.Interface) (*ArpConn, error) {
	var err error
	conn := &ArpConn{ifindex: ifi.Index, proto: binary.BigEndian.Uint16(binary.LittleEndian.AppendUint16(nil, ETH_P_ARP))}
	conn.Socket, err = socket.NewSocket(syscall.AF_PACKET, syscall.SOCK_RAW|syscall.SOCK_CLOEXEC|syscall.SOCK_NONBLOCK, int(conn.proto), name)
	if err != nil {
		return nil, err
	}
	lsa := &syscall.SockaddrLinklayer{Protocol: conn.proto, Ifindex: conn.ifindex}
	var sal syscall.Sockaddr
	if sal, err = conn.Bind(lsa); err != nil {
		conn.Close()
		return nil, err
	}
	lsa = sal.(*syscall.SockaddrLinklayer)
	l, _ := ifi.Addrs()
	conn.addr = net.ParseIP("0.0.0.0")
	conn.hardware = net.HardwareAddr(lsa.Addr[:])
	if 0 < len(l) {
		conn.addr = (l[0].(*net.IPNet)).IP
	}
	return conn, err
}

func (ac *ArpConn) Read(b []byte) (n int, err error) {
	n, _, err = ac.Recvfrom(b, 0)
	return
}

func (ac *ArpConn) Write(b []byte) error {
	lsa := &syscall.SockaddrLinklayer{Ifindex: ac.ifindex, Protocol: ac.proto, Halen: uint8(len(ac.hardware))}
	copy(lsa.Addr[:], ac.hardware)
	return ac.Sendto(b, 0, lsa)
}

func (ac *ArpConn) Request(ip net.IP) error {
	b := (&packet.ArpPacket{
		HeadMAC: [2]packet.HardwareAddr{packet.Broadcast, (packet.HardwareAddr)(ac.hardware)},
		FrameType: ETH_P_ARP, HardwareType: packet.ARP_ETHERNETTYPE, ProtocolType: ETH_P_IP,
		HardwareLen: 6, IPLen: 4, Operation: packet.ARP_REQUEST,
		SendHardware: ([6]byte)(ac.hardware),
		SendIP: ([4]byte)(ac.addr.To4()),
		TargetHardware: packet.Broadcast,
		TargetIP: ([4]byte)(ip.To4()),
	}).WireFormat()
	return ac.Write(b)
}

func (ac *ArpConn) GetAllIPMAC(out time.Duration) []*packet.ArpPacket {
	go func() {
		targetIP := ac.addr.To4()
		for i := 0; i < 255; i++ {
			targetIP[3] = byte(i)
			ac.Request(net.IP(targetIP))
		}
	}()
	var list []*packet.ArpPacket
	over, b := false, make([]byte, 128)
	timer := time.AfterFunc(out, func () { over = true; })
	for !over {
		n, err := ac.Read(b)
		if err != nil {
			break
		}
		p := packet.NewArpPacket(([42]byte)(b[:n]))
		if p.Operation == 2 {
			timer.Reset(out)
			list = append(list, p)
		}
	}
	return list
}
