// @@
// @ Author       : Eacher
// @ Date         : 2023-06-29 15:13:47
// @ LastEditTime : 2023-07-06 11:55:38
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /networkmanager/socket/dhcpv4/dhcpv4.go
// @@
package dhcpv4

import (
	"net"
	"time"
	"syscall"

	"github.com/20yyq/packet"
)

type DhcpV4Conn struct {
	*net.UDPConn
	server 	net.UDPAddr
	xid 	uint32
	rc 		syscall.RawConn
}

func NewDhcpV4Conn(name string, addr packet.IPv4) (*DhcpV4Conn, error) {
	listen, err := net.ListenUDP("udp4", &net.UDPAddr{net.IPv4bcast, packet.DHCP_ClientPort, name})
	if err != nil {
		return nil, err
	}
	conn := &DhcpV4Conn{listen, net.UDPAddr{net.IP(addr[:]), packet.DHCP_ServerPort, ""}, uint32(time.Now().Unix()), nil}
	conn.rc, err = conn.SyscallConn()
	if err != nil {
		conn.Close()
		return nil, err
	}
	fun := func(fd uintptr) {
		err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
	}
	err = conn.rc.Control(fun)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return conn, err
}

func (v4 *DhcpV4Conn) Discover() (offer *packet.DhcpV4Packet, err error) {
	count := 0
LOOP:
	count++
	offer = &packet.DhcpV4Packet{
		Op: packet.DHCP_BOOTREQUEST, HardwareType: packet.DHCP_Ethernet_TYPE, 
		HardwareLen: packet.DHCP_Ethernet_LEN, XID: v4.xid,
	}
	offer.Options = append(offer.Options, packet.SetDHCPMessage(packet.DHCP_DISCOVER), 
						packet.SetDHCPOptionsRequestList(1,3,6))
	if _, err := v4.WriteToUDPAddrPort(offer.WireFormat(), v4.server.AddrPort()); err != nil {
		return nil, err
	}
	b := make([]byte, 512)
	n, _, err := v4.ReadFromUDPAddrPort(b)
	if err != nil {
		return nil, err
	}
	offer = packet.NewDhcpV4Packet(b[:n])
	if offer.XID != v4.xid && 15 > count {
		time.Sleep(time.Second)
		goto LOOP
	}
	if offer.Op != packet.DHCP_BOOTREPLY {
		v4.xid = uint32(time.Now().Unix())
		return nil, err
	}
	return
}

func (v4 *DhcpV4Conn) Request(offer packet.DhcpV4Packet) (ack *packet.DhcpV4Packet, err error) {
	count := 0
LOOP:
	count++
	offer.Op = packet.DHCP_BOOTREQUEST
	offer.Options = offer.Options[1:]
	offer.Options = append(offer.Options, packet.SetDHCPMessage(packet.DHCP_REQUEST))
	if _, err := v4.WriteToUDPAddrPort(offer.WireFormat(), v4.server.AddrPort()); err != nil {
		return nil, err
	}
	b := make([]byte, 512)
	n, _, err := v4.ReadFromUDPAddrPort(b)
	if err != nil {
		return nil, err
	}
	ack = packet.NewDhcpV4Packet(b[:n])
	if ack.XID != v4.xid && 15 > count {
		time.Sleep(time.Second)
		goto LOOP
	}
	if ack.Op != packet.DHCP_BOOTREPLY {
		v4.xid = uint32(time.Now().Unix())
		return nil, err
	}
	v4.xid = uint32(time.Now().Unix())
	return
}
