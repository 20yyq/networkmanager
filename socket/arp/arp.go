// @@
// @ Author       : Eacher
// @ Date         : 2023-06-29 15:13:47
// @ LastEditTime : 2023-09-14 11:24:45
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /20yyq/networkmanager/socket/arp/arp.go
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

	control socket.RawConnControl
	lsa 	*syscall.SockaddrLinklayer
	addr 	net.IP
}

func NewArpConn(name string, ifi *net.Interface) (*ArpConn, error) {
	var err error
	conn, proto := &ArpConn{addr: net.ParseIP("0.0.0.0")}, binary.BigEndian.Uint16(binary.LittleEndian.AppendUint16(nil, ETH_P_ARP))
	conn.Socket, err = socket.NewSocket(syscall.AF_PACKET, syscall.SOCK_RAW|syscall.SOCK_CLOEXEC|syscall.SOCK_NONBLOCK, int(proto), name)
	if err != nil {
		return nil, err
	}
	conn.control, _ = conn.Socket.Control()
	fun := func(fd uintptr) {
		conn.lsa = &syscall.SockaddrLinklayer{Protocol: proto, Ifindex: ifi.Index}
		if err = syscall.Bind(int(fd), conn.lsa); err != nil {
			return
		}
		var sal syscall.Sockaddr
		if sal, err = syscall.Getsockname(int(fd)); err != nil {
			return
		}
		conn.lsa = sal.(*syscall.SockaddrLinklayer)
	}
	if e := conn.control(fun); e != nil {
		conn.Socket.Close()
		return nil, e
	}
	if err != nil {
		conn.Socket.Close()
		return nil, err
	}
	l, _ := ifi.Addrs()
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
	return ac.Sendto(b, 0, ac.lsa)
}

func (ac *ArpConn) Request(ip net.IP) error {
	arpp := &packet.ArpPacket{
		HeadMAC: [2]packet.HardwareAddr{packet.Broadcast, (packet.HardwareAddr)(ac.lsa.Addr[:ac.lsa.Halen])},
		FrameType: ETH_P_ARP, HardwareType: packet.ARP_ETHERNETTYPE, ProtocolType: ETH_P_IP,
		HardwareLen: 6, IPLen: 4, Operation: packet.ARP_REQUEST,
		SendHardware: ([6]byte)(ac.lsa.Addr[:ac.lsa.Halen]),
		SendIP: packet.IPv4{},
		TargetHardware: packet.Broadcast,
		TargetIP: ([4]byte)(ip.To4()),
	}
	if ac.addr.To4() != nil {
		arpp.SendIP = ([4]byte)(ac.addr.To4())
	}
	return ac.Write(arpp.WireFormat())
}

func (ac *ArpConn) GetAllIPMAC(out time.Duration) []*packet.ArpPacket {
	b := make([]byte, 128)
	n, _ := ac.Read(b)
	go func(p *packet.ArpPacket) {
		for i := 0; i < 255; i++ {
			p.SendIP[3] = byte(i)
			ac.Request(net.IP(p.SendIP[:]))
		}
	}(packet.NewArpPacket(([42]byte)(b[:n])))
	var list []*packet.ArpPacket
	over := false
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
