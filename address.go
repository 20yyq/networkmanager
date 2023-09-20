// @@
// @ Author       : Eacher
// @ Date         : 2023-06-27 09:39:36
// @ LastEditTime : 2023-09-20 11:33:22
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /20yyq/networkmanager/address.go
// @@
package networkmanager

import (
	"net"
	"unsafe"
	"syscall"

	"github.com/20yyq/packet"
	"github.com/20yyq/netlink"
)

type Addrs struct {
	*packet.IfAddrmsg
	label 		string
	address 	net.IP
	Local 		net.IP
	Broadcast 	net.IP
	Anycast 	net.IP

	Cache 		*cacheInfo
}

type cacheInfo struct {
	PreferredLft 	uint32
	ValidLft 		uint32
	CreadTime 		uint32
	UpdateTime 		uint32
}

func (ifi *Interface) IPList() ([]*Addrs, error) {
	var err error
	sm := netlink.SendNLMessage{
		NlMsghdr: &packet.NlMsghdr{Type: RTM_GETADDR, Flags: NLM_F_REQUEST|NLM_F_DUMP, Seq: randReq()},
	}
	sm.Attrs = append(sm.Attrs, packet.IfInfomsg{Family: AF_UNSPEC, Index: int32(ifi.iface.Index)})
	rm := netlink.ReceiveNLMessage{Data: make([]byte, 1024)}
	if err = ifi.conn.Exchange(&sm, &rm); err == nil {
		return ifi.deserializeIfAddrmsgMessages(&rm)
	}
	return nil, err
}

func (ifi *Interface) AddIP(a Addrs) error {
	sm := netlink.SendNLMessage{
		NlMsghdr: &packet.NlMsghdr{Type: RTM_NEWADDR, Flags: NLM_F_REQUEST|NLM_F_CREATE|NLM_F_EXCL|NLM_F_ACK, Seq: randReq()},
	}
	sm.Attrs = append(sm.Attrs, SerializeAddrs(&a, uint32(ifi.iface.Index))...)
	rm := netlink.ReceiveNLMessage{Data: make([]byte, 1024)}
	err := ifi.conn.Exchange(&sm, &rm)
	if err == nil {
		if rm.MsgList[0].Header.Type != RTM_NEWADDR {
			err = DeserializeNlMsgerr(rm.MsgList[0])
		}
	}
	return err
}

func (ifi *Interface) RemoveIP(a Addrs) error {
	sm := netlink.SendNLMessage{
		NlMsghdr: &packet.NlMsghdr{Type: RTM_DELADDR, Flags: NLM_F_REQUEST|NLM_F_ACK, Seq: randReq()},
	}
	sm.Attrs = append(sm.Attrs, SerializeAddrs(&a, uint32(ifi.iface.Index))...)
	rm := netlink.ReceiveNLMessage{Data: make([]byte, 1024)}
	err := ifi.conn.Exchange(&sm, &rm)
	if err == nil {
		if rm.MsgList[0].Header.Type != RTM_DELADDR {
			err = DeserializeNlMsgerr(rm.MsgList[0])
		}
	}
	return err
}

func (ifi *Interface) ReplaceIP(a *Addrs) error {
	if a.address != nil {
		if err := ifi.ReplaceIP(&Addrs{Local: a.address, Cache: &cacheInfo{PreferredLft: 1, ValidLft: 1}}); err != nil {
			return err
		}
	}
	sm := netlink.SendNLMessage{
		NlMsghdr: &packet.NlMsghdr{Type: RTM_NEWADDR, Flags: NLM_F_REQUEST|NLM_F_ACK|NLM_F_REPLACE, Seq: randReq()},
	}
	sm.Attrs = append(sm.Attrs, SerializeAddrs(a, uint32(ifi.iface.Index))...)
	rm := netlink.ReceiveNLMessage{Data: make([]byte, 1024)}
	err := ifi.conn.Exchange(&sm, &rm)
	if err == nil {
		if rm.MsgList[0].Header.Type != RTM_NEWADDR {
			err = DeserializeNlMsgerr(rm.MsgList[0])
		}
	}
	return err
}

func SerializeAddrs(a *Addrs, idx uint32) []packet.Attrs {
	family, attrs := AF_INET, []packet.Attrs{}
	localAddr := a.Local.To4()
	if len(localAddr) == 0 {
		family = AF_INET6
		localAddr = a.Local.To16()
	}
	mask := a.Local.DefaultMask()
	prefixlen, masklen := mask.Size()
	if a.IfAddrmsg == nil {
		a.IfAddrmsg = &packet.IfAddrmsg{}
	}
	a.Prefixlen, a.Family, a.Index = uint8(prefixlen), uint8(family), idx
	attrs = append(attrs, *a.IfAddrmsg)
	attrs = append(attrs, packet.RtAttr{&syscall.RtAttr{uint16(len(localAddr) + packet.SizeofRtAttr), IFA_LOCAL}, localAddr})
	if a.Broadcast == nil && prefixlen < 31 {
		broadcast := make([]byte, masklen/8)
		for i := range localAddr {
			broadcast[i] = localAddr[i] | ^mask[i]
		}
		a.Broadcast = net.IPv4(broadcast[0], broadcast[1], broadcast[2], broadcast[3])
	}
	if a.Broadcast != nil {
		attrs = append(attrs, packet.RtAttr{&syscall.RtAttr{uint16(len(a.Broadcast.To4()) + packet.SizeofRtAttr), IFA_BROADCAST}, a.Broadcast.To4()})
	}
	if a.Anycast != nil {
		attrs = append(attrs, packet.RtAttr{&syscall.RtAttr{uint16(len(a.Anycast.To4()) + packet.SizeofRtAttr), IFA_ANYCAST}, a.Anycast.To4()})
	}
	if a.label != "" {
		attrs = append(attrs, packet.RtAttr{&syscall.RtAttr{uint16(len([]byte(a.label)) + packet.SizeofRtAttr), IFA_LABEL}, []byte(a.label)})
	}
	if a.Cache != nil && (a.Cache.PreferredLft > 0 || a.Cache.ValidLft > 0) {
		cache := (*(*[16]byte)(unsafe.Pointer(a.Cache)))[:]
		attrs = append(attrs, packet.RtAttr{&syscall.RtAttr{uint16(len(cache) + packet.SizeofRtAttr), IFA_CACHEINFO}, cache})
	}
	return attrs
}
