// @@
// @ Author       : Eacher
// @ Date         : 2023-06-21 08:16:59
// @ LastEditTime : 2023-06-29 11:14:49
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
	iface := &Interface{list: make(map[uint32]*NetlinkMessage), closes: make(chan struct{}), sock: syscall.SockaddrNetlink{Family: AF_NETLINK}}
	var err error
	if iface.iface, err = net.InterfaceByName(ifname); err != nil {
		return nil, err
	}
	iface.fd, err = syscall.Socket(AF_NETLINK, SOCK_RAW|SOCK_CLOEXEC, NETLINK_ROUTE)
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
	nl, err := ifi.request(RTM_NEWLINK, NLM_F_ACK, 
		(&IfInfomsg{Family: AF_UNSPEC, Flags: IFF_UP, Change: IFF_UP, Index: int32(ifi.iface.Index)}).WireFormat())
	if err == nil {
		nl.mutex.Lock()
		if nl.wait {
			nl.cond.Wait()
			nl.wait = false
		}
		nl.mutex.Unlock()
		go ifi.deleteTimeOverNotify(nl)
		if err = DeserializeNlMsgerr(nl.Message[0]); err == nil {
			ifi.iface.Flags++
		}
	}
	return err
}

func (ifi *Interface) Down() error {
	if ifi.iface.Flags&0x01 != 1 {
		return nil
	}
	nl, err := ifi.request(RTM_NEWLINK, NLM_F_ACK, 
		(&IfInfomsg{Family: AF_UNSPEC, Change: IFF_UP, Index: int32(ifi.iface.Index)}).WireFormat())
	if err == nil {
		nl.mutex.Lock()
		if nl.wait {
			nl.cond.Wait()
			nl.wait = false
		}
		nl.mutex.Unlock()
		go ifi.deleteTimeOverNotify(nl)
		if err = DeserializeNlMsgerr(nl.Message[0]); err == nil {
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
	<-ifi.closes
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
					syscall.SetsockoptTimeval(ifi.fd, SOL_SOCKET, SO_RCVTIMEO, &syscall.Timeval{Sec: sec})
				}
				continue
			}
			close(ifi.closes)
			break
		}
		if nr < NLMSG_HDRLEN {
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
	req := NlMsghdr{
		Len: SizeofNlMsghdr + uint32(len(data)), Type: uint16(proto),
		Seq: ifi.req, Flags: NLM_F_REQUEST | uint16(flags),
	}
	b := make([]byte, req.Len)
	req.wireFormat(b[:SizeofNlMsghdr])
	copy(b[SizeofNlMsghdr:], data)
	ifi.list[ifi.req] = nl
	if ifi.rcvtime < 1 {
		ifi.rcvtime = 3
		syscall.SetsockoptTimeval(ifi.fd, SOL_SOCKET, SO_RCVTIMEO, &syscall.Timeval{Sec: ifi.rcvtime})
	}
	if err := syscall.Sendto(ifi.fd, b, 0, &ifi.sock); err != nil {
		return nil, err
	}
	return nl, nil
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
