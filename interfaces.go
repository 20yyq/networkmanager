// @@
// @ Author       : Eacher
// @ Date         : 2023-06-21 08:16:59
// @ LastEditTime : 2023-09-14 09:51:56
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /20yyq/networkmanager/interfaces.go
// @@
package networkmanager

import (
	"net"
	"sync"
	"syscall"
	"encoding/binary"

	"github.com/20yyq/packet"
	"github.com/20yyq/netlink"
)

const (
	SOCK_CLOEXEC 	= syscall.SOCK_CLOEXEC
	SOL_SOCKET 		= syscall.SOL_SOCKET
	SOCK_RAW 		= syscall.SOCK_RAW
	SO_RCVTIMEO 	= syscall.SO_RCVTIMEO
	NLMSG_HDRLEN 	= syscall.NLMSG_HDRLEN
	NETLINK_ROUTE 	= syscall.NETLINK_ROUTE

	AF_NETLINK 		= syscall.AF_NETLINK
	AF_UNSPEC 		= syscall.AF_UNSPEC
	AF_INET 		= syscall.AF_INET
	AF_INET6 		= syscall.AF_INET6

	IFA_LOCAL 		= syscall.IFA_LOCAL
	IFA_BROADCAST 	= syscall.IFA_BROADCAST
	IFA_ANYCAST 	= syscall.IFA_ANYCAST
	IFA_LABEL 		= syscall.IFA_LABEL
	IFA_CACHEINFO 	= syscall.IFA_CACHEINFO
	IFA_ADDRESS 	= syscall.IFA_ADDRESS
	IFF_UP 			= syscall.IFF_UP

	NLM_F_CREATE 	= syscall.NLM_F_CREATE
	NLM_F_REQUEST 	= syscall.NLM_F_REQUEST
	NLM_F_EXCL 		= syscall.NLM_F_EXCL
	NLM_F_ACK 		= syscall.NLM_F_ACK
	NLM_F_DUMP 		= syscall.NLM_F_DUMP
	NLM_F_REPLACE 	= syscall.NLM_F_REPLACE

	RTM_NEWADDR 	= syscall.RTM_NEWADDR
	RTM_GETADDR 	= syscall.RTM_GETADDR
	RTM_DELADDR 	= syscall.RTM_DELADDR
	RTM_NEWLINK 	= syscall.RTM_NEWLINK
	RTM_GETROUTE 	= syscall.RTM_GETROUTE
	RTM_NEWROUTE 	= syscall.RTM_NEWROUTE
	RTM_DELROUTE 	= syscall.RTM_DELROUTE

	RTA_DST 		= syscall.RTA_DST
	RTA_SRC 		= syscall.RTA_SRC
	RTA_PREFSRC 	= syscall.RTA_PREFSRC
	RTA_IIF 		= syscall.RTA_IIF
	RTA_OIF 		= syscall.RTA_OIF
	RTA_GATEWAY 	= syscall.RTA_GATEWAY
	RTA_PRIORITY 	= syscall.RTA_PRIORITY
	RTA_METRICS 	= syscall.RTA_METRICS
	RTA_FLOW 		= syscall.RTA_FLOW
	RTA_TABLE 		= syscall.RTA_TABLE
	RTA_CACHEINFO 	= syscall.RTA_CACHEINFO

	RT_TABLE_MAIN 	= syscall.RT_TABLE_MAIN
	RT_SCOPE_UNIVERSE = syscall.RT_SCOPE_UNIVERSE
	RTPROT_BOOT 	= syscall.RTPROT_BOOT
	RTN_UNICAST 	= syscall.RTN_UNICAST

	RTNLGRP_ND_USEROPT = syscall.RTNLGRP_ND_USEROPT

	SizeofRtAttr 	= syscall.SizeofRtAttr
)

type Interface struct {
	iface	*net.Interface
	conn 	*netlink.NetlinkRoute
	
	mutex 	sync.Mutex
	req		uint32
}

func InterfaceByName(ifname string) (*Interface, error) {
	iface := &Interface{}
	var err error
	if iface.iface, err = net.InterfaceByName(ifname); err != nil {
		return nil, err
	}
	iface.conn = &netlink.NetlinkRoute{
		DevName: iface.iface.Name,
		Sal: &syscall.SockaddrNetlink{Family: syscall.AF_NETLINK, Groups: syscall.RTNLGRP_LINK},
	}
	return iface, iface.conn.Init() 
}

func (ifi *Interface) Up() error {
	if ifi.iface.Flags&0x01 != 0 {
		return nil
	}
	sm := netlink.SendNLMessage{
		NlMsghdr: &packet.NlMsghdr{Type: RTM_NEWLINK, Flags: syscall.NLM_F_REQUEST|NLM_F_ACK, Seq: ifi.req},
		Data: (&packet.IfInfomsg{Family: AF_UNSPEC, Flags: IFF_UP, Change: IFF_UP, Index: int32(ifi.iface.Index)}).WireFormat(),
	}
	sm.Len = packet.SizeofNlMsghdr + uint32(len(sm.Data))
	rm := netlink.ReceiveNLMessage{Data: make([]byte, 128)}
	err := ifi.conn.Exchange(&sm, &rm)
	if err == nil {
		if err = DeserializeNlMsgerr(rm.MsgList[0]); err == nil {
			ifi.iface.Flags++
		}
	}
	return err
}

func (ifi *Interface) Down() error {
	if ifi.iface.Flags&0x01 != 1 {
		return nil
	}
	sm := netlink.SendNLMessage{
		NlMsghdr: &packet.NlMsghdr{Type: RTM_NEWLINK, Flags: syscall.NLM_F_REQUEST|NLM_F_ACK, Seq: ifi.req},
		Data: (&packet.IfInfomsg{Family: AF_UNSPEC, Change: IFF_UP, Index: int32(ifi.iface.Index)}).WireFormat(),
	}
	sm.Len = packet.SizeofNlMsghdr + uint32(len(sm.Data))
	rm := netlink.ReceiveNLMessage{Data: make([]byte, 128)}
	err := ifi.conn.Exchange(&sm, &rm)
	if err == nil {
		if err = DeserializeNlMsgerr(rm.MsgList[0]); err == nil {
			ifi.iface.Flags--
		}
	}
	return err
}

func (ifi *Interface) Close() {
	ifi.mutex.Lock()
	defer ifi.mutex.Unlock()
	ifi.conn.Close()
}

func RtAttrToSliceByte(types uint16, ip net.IP, ips ...net.IP) []byte {
	var children []byte
	if len(ips) > 0 {
		nextIP, nextIps := ips[0], ips[1:]
		children = RtAttrToSliceByte(types, nextIP, nextIps...)
	}
	l, next := uint16(SizeofRtAttr + len(ip) + len(children)), 0
	data := make([]byte, l)
	binary.LittleEndian.PutUint16(data[next:], l)
	next += 2
	binary.LittleEndian.PutUint16(data[next:], types)
	next += 2
	copy(data[next:], ip)
	next += len(ip)
	copy(data[next:], children)
	return data
}

func appendSliceByte(data []byte, types uint16, ips ...net.IP) []byte {
	tmp := data
	if 0 < len(ips) {
		res := RtAttrToSliceByte(types, ips[0], ips[1:]...)
		data = make([]byte, len(tmp) + len(res))
		copy(data[:len(tmp)], tmp)
		copy(data[len(tmp):], res)
	}
	return data
}
