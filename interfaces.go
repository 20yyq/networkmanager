// @@
// @ Author       : Eacher
// @ Date         : 2023-06-21 08:16:59
// @ LastEditTime : 2023-06-21 14:15:36
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /networkmanager/interfaces.go
// @@
package networkmanager

import (
	"fmt"
	"net"
	"time"
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

type NetlinkMessage struct {
	req 	uint32
	pid 	uint32
	err 	error
	wait 	bool
	mutex 	sync.Mutex
	cond	*sync.Cond
	
	Message []*syscall.NetlinkMessage
}

func (nlm *NetlinkMessage) serializeMessages(label string) *Addrs {
	var attr *syscall.RtAttr
	for _, m := range nlm.Message {
		if len(m.Data) < syscall.SizeofIfAddrmsg {
			break
		}
		addrsingle := Addrs{addr: (*syscall.IfAddrmsg)(unsafe.Pointer(&m.Data[:syscall.SizeofIfAddrmsg][0]))}
		tmp := m.Data[syscall.SizeofIfAddrmsg:]
		for len(tmp) >= syscall.SizeofRtAttr {
			attr = (*syscall.RtAttr)(unsafe.Pointer(&tmp[0]))
			if attr.Len < syscall.SizeofRtAttr || int(attr.Len) > len(tmp) {
				break
			}
			switch attr.Type {
			case syscall.IFA_ADDRESS:
				log.Println("syscall.IFA_ADDRESS")
				addrsingle.Netmask = net.IPMask(tmp[syscall.SizeofRtAttr:attr.Len])
			case syscall.IFA_LOCAL:
				log.Println("syscall.IFA_LOCAL")
				addrsingle.Local = net.IP(tmp[syscall.SizeofRtAttr:attr.Len])
			case syscall.IFA_BROADCAST:
				log.Println("syscall.IFA_BROADCAST")
				addrsingle.Broadcast = net.IP(tmp[syscall.SizeofRtAttr:attr.Len])
			case syscall.IFA_LABEL:
				addrsingle.label = string(tmp[syscall.SizeofRtAttr:(attr.Len - 1)])
				log.Println("syscall.IFA_LABEL", addrsingle.label)
			case syscall.IFLA_COST:
				log.Println("syscall.IFLA_COST", string(tmp[syscall.SizeofRtAttr:attr.Len]), tmp[syscall.SizeofRtAttr:attr.Len])
				// addrsingle.Broadcast = net.IP(tmp[syscall.SizeofRtAttr:attr.Len])
				// addr.Label = string(attr.Value[:len(attr.Value)-1])
			// case IFA_FLAGS:
			// 	log.Println("syscall.IFA_FLAGS")
			// 	addr.Flags = int(native.Uint32(attr.Value[0:4]))
			case syscall.IFA_CACHEINFO:
				log.Println("syscall.IFA_CACHEINFO")
			// 	ci := nl.DeserializeIfaCacheInfo(attr.Value)
			// 	addr.PreferedLft = int(ci.IfaPrefered)
			// 	addr.ValidLft = int(ci.IfaValid)
			}
			// routeAttr := syscall.NetlinkRouteAttr{Attr: *attr, Value: tmp[:int(attr.Len - syscall.SizeofRtAttr)]}
			// res = append(attrs, ra)
			tmp = tmp[attr.Len:]
		}
		if addrsingle.label == label {
			return &addrsingle
		}
	}
	return nil
}


type Addrs struct {
	addr 		*syscall.IfAddrmsg
	label 		string
	Local 		net.IP
	Broadcast 	net.IP
	Netmask 	net.IPMask
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

func (ifi *Interface) IPList() (*Addrs, error) {
	var res *Addrs
	nl, err := ifi.request(syscall.RTM_GETADDR, syscall.NLM_F_DUMP, 
		IfInfomsgToSliceByte(syscall.AF_UNSPEC, 0x00, 0x00, int32(ifi.iface.Index)))
	if err == nil {
	Loop:
		nl.mutex.Lock()
		if nl.wait {
			nl.cond.Wait()
		}
		nl.wait, err = false, nl.err
		nl.mutex.Unlock()
		if res = nl.serializeMessages(ifi.iface.Name); res == nil && err == nil {
			nl.wait = true
			goto Loop
		}
		go ifi.deleteTimeOverNotify(nl)
	}
	return res, err

}

func (ifi *Interface) Up() error {
	if ifi.iface.Flags&0x01 == 0 {
		nl, err := ifi.request(syscall.RTM_NEWLINK, syscall.NLM_F_ACK, 
			IfInfomsgToSliceByte(syscall.AF_UNSPEC, syscall.IFF_UP, syscall.IFF_UP, int32(ifi.iface.Index)))
		if err == nil {
			nl.mutex.Lock()
			if nl.wait {
				nl.cond.Wait()
			}
			nl.wait, err = false, nl.err
			nl.mutex.Unlock()
			go ifi.deleteTimeOverNotify(nl)
			ifi.iface.Flags++
		}
	}
	return nil
}

func (ifi *Interface) Down() error {
	if ifi.iface.Flags&0x01 == 1 {
		nl, err := ifi.request(syscall.RTM_NEWLINK, syscall.NLM_F_ACK, 
			IfInfomsgToSliceByte(syscall.AF_UNSPEC, syscall.IFF_UP, 0x00, int32(ifi.iface.Index)))
		if err == nil {
			nl.mutex.Lock()
			if nl.wait {
				nl.cond.Wait()
			}
			nl.wait, err = false, nl.err
			nl.mutex.Unlock()
			go ifi.deleteTimeOverNotify(nl)
			ifi.iface.Flags--
		}
	}
	return nil
}

func (ifi *Interface) AddIP(a Addrs) error {
	data := SerializeAddrs(&a, uint32(ifi.iface.Index))
	nl, err := ifi.request(syscall.RTM_NEWADDR, syscall.NLM_F_CREATE|syscall.NLM_F_EXCL|syscall.NLM_F_ACK, data)
	if err == nil {
		nl.mutex.Lock()
		if nl.wait {
			nl.cond.Wait()
		}
		nl.wait, err = false, nl.err
		nl.mutex.Unlock()
		go ifi.deleteTimeOverNotify(nl)
	}
	return err
}

func (ifi *Interface) RemoveIP(a Addrs) error {
	data := SerializeAddrs(&a, uint32(ifi.iface.Index))
	nl, err := ifi.request(syscall.RTM_DELADDR, syscall.NLM_F_ACK, data)
	if err == nil {
		nl.mutex.Lock()
		if nl.wait {
			nl.cond.Wait()
		}
		nl.wait, err = false, nl.err
		nl.mutex.Unlock()
		go ifi.deleteTimeOverNotify(nl)
	}
	return err
}

func (ifi *Interface) ReplaceIP(a Addrs) error {
	tmp := SerializeAddrs(&a, uint32(ifi.iface.Index))
	cache := make([]byte, 8)
	binary.BigEndian.PutUint32(cache[0:], 300)
	binary.BigEndian.PutUint32(cache[4:], 300)
	res := RtAttrToSliceByte(syscall.IFA_CACHEINFO, cache)
	data := make([]byte, len(tmp) + len(res))
	copy(data[:len(tmp)], tmp)
	copy(data[len(tmp):], res)
	nl, err := ifi.request(syscall.RTM_NEWADDR, syscall.NLM_F_CREATE|syscall.NLM_F_REPLACE|syscall.NLM_F_ACK, data)
	if err == nil {
		nl.mutex.Lock()
		if nl.wait {
			nl.cond.Wait()
		}
		nl.wait, err = false, nl.err
		nl.mutex.Unlock()
		go ifi.deleteTimeOverNotify(nl)
	}
	return err
}

func SerializeAddrs(a *Addrs, idx uint32) []byte {
	family := syscall.AF_INET
	if len(a.Local) == net.IPv6len {
		family = syscall.AF_INET6
	}
	prefixlen, masklen := a.Local.DefaultMask().Size()
	res := IfAddrmsgToSliceByte(uint8(family), uint8(prefixlen), 0x00, 0x00, idx)
	localAddr := a.Local.To4()
	if family == syscall.AF_INET6 {
		localAddr = a.Local.To16()
	}
	data := make([]byte, len(res))
	copy(data, res)
	tmp := data
	res = RtAttrToSliceByte(syscall.IFA_LOCAL, localAddr)
	data = make([]byte, len(tmp) + len(res))
	copy(data[:len(tmp)], tmp)
	copy(data[len(tmp):], res)
	tmp = data
	if a.Netmask == nil {
		a.Netmask = a.Local.DefaultMask()
	}
	res = RtAttrToSliceByte(syscall.IFA_ADDRESS, []byte(a.Netmask))
	data = make([]byte, len(tmp) + len(res))
	copy(data[:len(tmp)], tmp)
	copy(data[len(tmp):], res)
	tmp = data
	if a.Broadcast == nil && prefixlen < 31 {
		broadcast := make(net.IP, masklen/8)
		for i := range localAddr {
			broadcast[i] = localAddr[i] | ^a.Local.DefaultMask()[i]
		}
		a.Broadcast = broadcast
	}
	if a.Broadcast != nil {
		res = RtAttrToSliceByte(syscall.IFA_BROADCAST, a.Broadcast)
		data = make([]byte, len(tmp) + len(res))
		copy(data[:len(tmp)], tmp)
		copy(data[len(tmp):], res)
		tmp = data
	}
	log.Println("SerializeAddrs data", data, a.Broadcast, prefixlen, len(a.Local))
	return data
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

func (ifi *Interface) Close() {
	ifi.mutex.Lock()
	defer ifi.mutex.Unlock()
	if err := syscall.Close(ifi.fd); err != nil {
		log.Println("syscall.Close error", err, ifi.fd)
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
					ifi.lMutex.RLock()
					for _, l := range ifi.list {
						l.mutex.Lock()
						if l.wait {
							l.cond.Signal()
						}
						l.err = fmt.Errorf("resource temporarily unavailable")
						l.mutex.Unlock()
					}
					ifi.lMutex.RUnlock()
					syscall.SetsockoptTimeval(ifi.fd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &syscall.Timeval{Sec: sec})
				}
				continue
			}
			log.Println("Got short response from netlink", err)
			close(ifi.closes)
			break
		}
		if nr < syscall.NLMSG_HDRLEN {
			continue
		}
		if sock, _ := from.(*syscall.SockaddrNetlink); sock == nil || sock.Pid != ifi.sock.Pid {
			log.Println("Error converting to netlink sockaddr")
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
					log.Println("v.Data ", v.Header.Seq, string(v.Data), binary.LittleEndian.Uint32(v.Data[:4]))
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
