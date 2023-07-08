// @@
// @ Author       : Eacher
// @ Date         : 2023-06-29 15:13:47
// @ LastEditTime : 2023-07-08 16:12:33
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /networkmanager/socket/rtnetlink/rtnetlink.go
// @@
package rtnetlink

import (
	"os"
	"net"
	"time"
	"syscall"
	"sync"
	"sync/atomic"

	"github.com/20yyq/packet"
	"github.com/20yyq/networkmanager/socket"
)

type NetlinkMessage struct {
	req 	uint32
	wait 	time.Duration
	timer	*time.Timer

	Notify 	<-chan struct{}
	Message []*syscall.NetlinkMessage
}

type RtnetlinkConn struct {
	*socket.Socket

	req 	uint32
	list 	map[uint32]*NetlinkMessage
	mutex 	sync.RWMutex
	lsa 	*syscall.SockaddrNetlink
	closes  chan struct{}
	control socket.RawConnControl
}

func NewRtnetlinkConn(dev string, ifi *net.Interface) (*RtnetlinkConn, error) {
	var err error
	conn := &RtnetlinkConn{list: make(map[uint32]*NetlinkMessage), closes: make(chan struct{})}
	conn.Socket, err = socket.NewSocket(syscall.AF_NETLINK, syscall.SOCK_RAW|syscall.SOCK_CLOEXEC, syscall.NETLINK_ROUTE, dev)
	if err != nil {
		return nil, err
	}
	conn.control, _ = conn.Socket.Control()
	fun := func(fd uintptr) {
		if err = syscall.BindToDevice(int(fd), dev); err != nil {
			return
		}
		conn.lsa = &syscall.SockaddrNetlink{Family: syscall.AF_NETLINK, Groups: syscall.RTNLGRP_LINK}
		if err = syscall.Bind(int(fd), conn.lsa); err != nil {
			return
		}
	}
	if e := conn.control(fun); e != nil {
		conn.Socket.Close()
		return nil, e
	}
	if err != nil {
		conn.Socket.Close()
		return nil, err
	}
	go conn.receive()
	return conn, err
}

func (rt *RtnetlinkConn) Close() {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()
	if err := rt.Socket.Close(); err != nil {
		return
	}
	<-rt.closes
}

func (rt *RtnetlinkConn) Exchange(wait, proto, flags uint16, data []byte) (*NetlinkMessage, error) {
	nl := &NetlinkMessage{req: atomic.AddUint32(&rt.req, 1), wait: time.Duration(wait)*time.Millisecond}
	req := packet.NlMsghdr{
		Len: packet.SizeofNlMsghdr + uint32(len(data)), Type: proto,
		Seq: nl.req, Flags: syscall.NLM_F_REQUEST|flags,
	}
	b := make([]byte, req.Len)
	req.WireFormatToByte((*[packet.SizeofNlMsghdr]byte)(b))
	copy(b[packet.SizeofNlMsghdr:], data)
	rt.mutex.Lock()
	rt.list[nl.req] = nl
	rt.mutex.Unlock()
	notify := make(chan struct{})
	nl.Notify = notify
	go rt.deleteNlm(nl, notify)
	if err := rt.Sendto(b, 0, rt.lsa); err != nil {
		return nil, err
	}
	return nl, nil
}

func (rt *RtnetlinkConn) deleteNlm(nlm *NetlinkMessage, notify chan struct{}) {
	fun := func () {
		rt.mutex.Lock()
		delete(rt.list, nlm.req)
		rt.mutex.Unlock()
		close(notify)
	}
	nlm.timer = time.AfterFunc(nlm.wait, fun)
}

func (rt *RtnetlinkConn) receive() {
	var read [4096]byte
	for {
		n, _, err := rt.Recvfrom(read[:], 0)
		if err != nil {
			if os.IsTimeout(err) {
				continue
			}
			rt.Socket.Close()
			close(rt.closes)
			break
		}
		if n < syscall.NLMSG_HDRLEN {
			continue
		}
		b := make([]byte, n)
		copy(b, read[:n])
		if nl, _ := syscall.ParseNetlinkMessage(b); 0 < len(nl) {
			go func(nl []syscall.NetlinkMessage) {
				notifyList := make(map[uint32]*NetlinkMessage)
				rt.mutex.RLock()
				for _, v := range nl {
					tmp := v
					if l, _ := rt.list[tmp.Header.Seq]; tmp.Header.Seq > 0 && l != nil {
						l.Message = append(l.Message, &tmp)
						if val, _ := notifyList[tmp.Header.Seq]; val == nil {
							notifyList[tmp.Header.Seq] = l
						}
						continue
					}
				}
				rt.mutex.RUnlock()
				for _, val := range notifyList {
					if val.timer != nil {
						wait := val.wait
						if wait > 100*time.Millisecond {
							wait /= 500*time.Microsecond
						}
						val.timer.Reset(wait)
					}
				}
			}(nl)
		}
	}
}
