// @@
// @ Author       : Eacher
// @ Date         : 2023-06-21 08:16:59
// @ LastEditTime : 2023-06-27 09:54:38
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /networkmanager/interfaces.go
// @@
package networkmanager

import (
	"net"
	"sync"
	"unsafe"
	"syscall"
	"encoding/binary"
)

type Interface struct {
	fd 		int
	pid 	uint32
	req 	uint32
	rcvtime int64
	iface	*net.Interface
	sock 	syscall.SockaddrNetlink
	list 	map[uint32]*NetlinkMessage
	closes  chan struct{}
	
	mutex 	sync.Mutex
	lMutex 	sync.RWMutex
}

func InterfaceByName(ifname string) (*Interface, error) {
	iface := &Interface{list: make(map[uint32]*NetlinkMessage), closes: make(chan struct{}), sock: syscall.SockaddrNetlink{Family: syscall.AF_NETLINK}}
	var err error
	if iface.iface, err = net.InterfaceByName(ifname); err != nil {
		return nil, err
	}
	iface.fd, err = syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_RAW|syscall.SOCK_CLOEXEC, syscall.NETLINK_ROUTE)
	if err != nil {
		return nil, err
	}
	if err = syscall.Bind(iface.fd, &iface.sock); err != nil {
		syscall.Close(iface.fd)
		return nil, err
	}
	iface.pid = uint32(syscall.Getpid())
	go iface.receive()
	return iface, err 
}

func (ifi *Interface) deleteTimeOverNotify(nl *NetlinkMessage) {
	ifi.lMutex.Lock()
	defer ifi.lMutex.Unlock()
	delete(ifi.list, nl.req)
}

func (ifi *Interface) Up() error {
	if ifi.iface.Flags&0x01 != 0 {
		return nil
	}
	nl, err := ifi.request(syscall.RTM_NEWLINK, syscall.NLM_F_ACK, 
		IfInfomsgToSliceByte(syscall.AF_UNSPEC, syscall.IFF_UP, syscall.IFF_UP, int32(ifi.iface.Index)))
	if err == nil {
		nl.mutex.Lock()
		if nl.wait {
			nl.cond.Wait()
		}
		nl.wait = false
		nl.mutex.Unlock()
		go ifi.deleteTimeOverNotify(nl)
		err = DeserializeNlMsgerr(nl.Message[0])
		if err == nil {
			ifi.iface.Flags++
		}
	}
	return err
}

func (ifi *Interface) Down() error {
	if ifi.iface.Flags&0x01 != 1 {
		return nil
	}
	nl, err := ifi.request(syscall.RTM_NEWLINK, syscall.NLM_F_ACK, 
		IfInfomsgToSliceByte(syscall.AF_UNSPEC, syscall.IFF_UP, 0x00, int32(ifi.iface.Index)))
	if err == nil {
		nl.mutex.Lock()
		if nl.wait {
			nl.cond.Wait()
		}
		nl.wait = false
		nl.mutex.Unlock()
		go ifi.deleteTimeOverNotify(nl)
		err = DeserializeNlMsgerr(nl.Message[0])
		if err == nil {
			ifi.iface.Flags--
		}
	}
	return err
}

func (ifi *Interface) Close() {
	ifi.mutex.Lock()
	defer ifi.mutex.Unlock()
	if err := syscall.Close(ifi.fd); err != nil {
		return
	}
	ifi.fd = -1
}

func (ifi *Interface) receive() {
	var read [4096]byte
	for {
		nr, from, err := syscall.Recvfrom(ifi.fd, read[:], 0)
		if err != nil {
			if err.(syscall.Errno).Temporary() {
				if ifi.rcvtime--; ifi.rcvtime < 0 {
					var sec int64 = ifi.rcvtime * -10
					if sec < 1 {
						sec = 0xFFFFFFFFFFFFFFF
					}
					syscall.SetsockoptTimeval(ifi.fd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &syscall.Timeval{Sec: sec})
				}
				continue
			}
			close(ifi.closes)
			break
		}
		if nr < syscall.NLMSG_HDRLEN {
			continue
		}
		if sock, _ := from.(*syscall.SockaddrNetlink); sock == nil || sock.Pid != ifi.sock.Pid {
			continue
		}
		rb2 := make([]byte, nr)
		copy(rb2, read[:nr])
		if nl, _ := syscall.ParseNetlinkMessage(rb2); 0 < len(nl) {
			go func(nl []syscall.NetlinkMessage) {
				notifyList := make(map[uint32]*NetlinkMessage)
				ifi.lMutex.RLock()
				for _, v := range nl {
					tmpv := v
					if l, _ := ifi.list[tmpv.Header.Seq]; l != nil {
						l.Message = append(l.Message, &tmpv)
						if val, _ := notifyList[tmpv.Header.Seq]; val == nil {
							notifyList[tmpv.Header.Seq] = l
						}
					}
				}
				ifi.lMutex.RUnlock()
				for _, val := range notifyList {
					val.mutex.Lock()
					if val.wait {
						val.cond.Signal()
					}
					val.mutex.Unlock()
				}
			}(nl)
		}
	}
}

func (ifi *Interface) request(proto, flags int, data []byte) (*NetlinkMessage, error) {
	ifi.mutex.Lock()
	defer ifi.mutex.Unlock()
	ifi.req++
	nl := &NetlinkMessage{req: ifi.req, pid: ifi.pid, wait: true}
	nl.cond = sync.NewCond(&nl.mutex)
	req := syscall.NlMsghdr{
		Len: uint32(syscall.SizeofNlMsghdr) + uint32(len(data)), Type: uint16(proto),
		Seq: ifi.req, Flags: syscall.NLM_F_REQUEST | uint16(flags),
	}
	b, hdr := make([]byte, req.Len), (*(*[syscall.SizeofNlMsghdr]byte)(unsafe.Pointer(&req)))[:]
	copy(b[:syscall.SizeofNlMsghdr], hdr)
	copy(b[syscall.SizeofNlMsghdr:], data)
	ifi.list[ifi.req] = nl
	if ifi.rcvtime < 1 {
		ifi.rcvtime = 3
		syscall.SetsockoptTimeval(ifi.fd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &syscall.Timeval{Sec: ifi.rcvtime})
	}
	if err := syscall.Sendto(ifi.fd, b, 0, &ifi.sock); err != nil {
		return nil, err
	}
	return nl, nil
}

func IfInfomsgToSliceByte(family uint8, change, flags uint32, idx int32) []byte {
	msg := &syscall.IfInfomsg{
		Family: family, 
		Change: change,
		Flags: flags,
		Index: idx,
	}
	return (*(*[syscall.SizeofIfInfomsg]byte)(unsafe.Pointer(msg)))[:]
}

func IfAddrmsgToSliceByte(family, prefixlen, flags, scope uint8, idx uint32) []byte {
	msg := &syscall.IfAddrmsg{
		Family: family, 
		Prefixlen: prefixlen,
		Flags: flags,
		Scope: scope,
		Index: idx,
	}
	return (*(*[syscall.SizeofIfAddrmsg]byte)(unsafe.Pointer(msg)))[:]
}

func RtMsgToSliceByte(family, dstlen, srclen, tos, table, protocol, scope, types uint8, flags uint32) []byte {
	msg := &syscall.RtMsg{
		Family:	family,
		Dst_len:dstlen,
		Src_len:srclen,
		Tos:	tos,
		Table:	table,
		Protocol:protocol,
		Scope:	scope,
		Type:	types,
		Flags:	flags,
	}
	return (*(*[syscall.SizeofRtMsg]byte)(unsafe.Pointer(msg)))[:]
}

func RtAttrToSliceByte(types uint16, ip net.IP, ips ...net.IP) []byte {
	var children []byte
	if len(ips) > 0 {
		nextIP, nextIps := ips[0], ips[1:]
		children = RtAttrToSliceByte(types, nextIP, nextIps...)
	}
	l, next := uint16(syscall.SizeofRtAttr + len(ip) + len(children)), 0
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
