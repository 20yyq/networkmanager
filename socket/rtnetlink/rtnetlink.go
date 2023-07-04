// @@
// @ Author       : Eacher
// @ Date         : 2023-06-29 15:13:47
// @ LastEditTime : 2023-07-04 15:13:11
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
	wait 	bool
	mutex 	sync.Mutex
	cond	*sync.Cond
	
	Message []*syscall.NetlinkMessage
}

type RtnetlinkConn struct {
	*socket.Socket

	req 	uint32
	list 	map[uint32]*NetlinkMessage
	mutex 	sync.RWMutex
	lsa 	*syscall.SockaddrNetlink
	closes  chan struct{}
}

func NewRtnetlinkConn(name string, ifi *net.Interface) (*RtnetlinkConn, error) {
	var err error
	conn := &RtnetlinkConn{list: make(map[uint32]*NetlinkMessage), closes: make(chan struct{})}
	conn.Socket, err = socket.NewSocket(syscall.AF_NETLINK, syscall.SOCK_RAW|syscall.SOCK_CLOEXEC, syscall.NETLINK_ROUTE, name)
	if err != nil {
		return nil, err
	}
	conn.lsa = &syscall.SockaddrNetlink{Family: syscall.AF_NETLINK}
	if _, err = conn.Bind(conn.lsa); err != nil {
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

func (rt *RtnetlinkConn) Exchange(proto, flags uint16, data []byte) (*NetlinkMessage, error) {
	nl := &NetlinkMessage{req: atomic.AddUint32(&rt.req, 1), wait: true}
	nl.cond = sync.NewCond(&nl.mutex)
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
	if err := rt.Sendto(b, 0, rt.lsa); err != nil {
		return nil, err
	}
	nl.mutex.Lock()
	if nl.wait {
		nl.cond.Wait()
		nl.wait = false
	}
	nl.mutex.Unlock()
	go rt.deleteNotify(nl)
	return nl, nil
}

func (rt *RtnetlinkConn) deleteNotify(nl *NetlinkMessage) {
	time.Sleep(time.Millisecond*150)
	rt.mutex.Lock()
	delete(rt.list, nl.req)
	rt.mutex.Unlock()
	nl.mutex.Lock()
	if nl.wait {
		nl.cond.Signal()
	}
	nl.mutex.Unlock()
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
					if l, _ := rt.list[v.Header.Seq]; l != nil {
						l.Message = append(l.Message, &v)
						if val, _ := notifyList[v.Header.Seq]; val == nil {
							notifyList[v.Header.Seq] = l
						}
					}
				}
				rt.mutex.RUnlock()
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
