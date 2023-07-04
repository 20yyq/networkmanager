// @@
// @ Author       : Eacher
// @ Date         : 2023-06-29 15:13:47
// @ LastEditTime : 2023-07-04 09:36:43
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /networkmanager/socket/dhcpv4/dhcpv4.go
// @@
package dhcpv4

import (
	"net"
	"syscall"

	"github.com/20yyq/packet"
)

type DhcpV4Conn struct {
	*net.UDPConn
	rc 	syscall.RawConn
}

func NewDhcpV4Conn(name string, ifi *net.Interface) (*DhcpV4Conn, error) {
	listen, err := net.ListenUDP("udp4", &net.UDPAddr{net.IPv4bcast, packet.DHCP_ClientPort, name})
	if err != nil {
		return nil, err
	}
	conn := &DhcpV4Conn{listen, nil}
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
