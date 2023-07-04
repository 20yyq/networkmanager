// @@
// @ Author       : Eacher
// @ Date         : 2023-06-27 09:39:36
// @ LastEditTime : 2023-07-04 15:15:12
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /networkmanager/address.go
// @@
package networkmanager

import (
	"net"
	"time"
	"unsafe"

	"github.com/20yyq/packet"
	"github.com/20yyq/networkmanager/socket/rtnetlink"
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
	var res []*Addrs
	var nl *rtnetlink.NetlinkMessage
	var err error
	count, wait := 0, false
Loop:
	nl, err = ifi.conn.Exchange(RTM_GETADDR, NLM_F_DUMP, (&packet.IfInfomsg{Family: AF_UNSPEC, Index: int32(ifi.iface.Index)}).WireFormat())
	if err == nil {
		count++
		time.Sleep(time.Millisecond*50)
		if res, err = ifi.deserializeIfAddrmsgMessages(nl); res == nil && err == nil {
			wait = true
		}
		if wait && 3 > count {
			goto Loop
		}
	}
	return res, err
}

func (ifi *Interface) AddIP(a Addrs) error {
	data := SerializeAddrs(&a, uint32(ifi.iface.Index))
	nl, err := ifi.conn.Exchange(RTM_NEWADDR, NLM_F_CREATE|NLM_F_EXCL|NLM_F_ACK, data)
	if err == nil {
		err = DeserializeNlMsgerr(nl.Message[0])
	}
	return err
}

func (ifi *Interface) RemoveIP(a Addrs) error {
	data := SerializeAddrs(&a, uint32(ifi.iface.Index))
	nl, err := ifi.conn.Exchange(RTM_DELADDR, NLM_F_ACK, data)
	if err == nil {
		err = DeserializeNlMsgerr(nl.Message[0])
	}
	return err
}

func (ifi *Interface) ReplaceIP(a *Addrs) error {
	if a.address != nil {
		if err := ifi.ReplaceIP(&Addrs{Local: a.address, Cache: &cacheInfo{PreferredLft: 1, ValidLft: 1}}); err != nil {
			return err
		}
	}
	data := SerializeAddrs(a, uint32(ifi.iface.Index))
	nl, err := ifi.conn.Exchange(RTM_NEWADDR, NLM_F_ACK|NLM_F_REPLACE, data)
	if err == nil {
		err = DeserializeNlMsgerr(nl.Message[0])
	}
	return err
}

func SerializeAddrs(a *Addrs, idx uint32) []byte {
	family := AF_INET
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
	data := a.WireFormat()
	data = appendSliceByte(data, IFA_LOCAL, localAddr)
	if a.Broadcast == nil && prefixlen < 31 {
		broadcast := make([]byte, masklen/8)
		for i := range localAddr {
			broadcast[i] = localAddr[i] | ^mask[i]
		}
		a.Broadcast = net.IPv4(broadcast[0], broadcast[1], broadcast[2], broadcast[3])
	}
	if a.Broadcast != nil {
		data = appendSliceByte(data, IFA_BROADCAST, a.Broadcast.To4())
	}
	if a.Anycast != nil {
		data = appendSliceByte(data, IFA_ANYCAST, a.Anycast.To4())
	}
	if a.label != "" {
		data = appendSliceByte(data, IFA_LABEL, []byte(a.label))
	}
	if a.Cache != nil && (a.Cache.PreferredLft > 0 || a.Cache.ValidLft > 0) {
		data = appendSliceByte(data, IFA_CACHEINFO, (*(*[16]byte)(unsafe.Pointer(a.Cache)))[:])
	}
	return data
}
