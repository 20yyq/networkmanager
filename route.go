// @@
// @ Author       : Eacher
// @ Date         : 2023-06-27 09:38:13
// @ LastEditTime : 2023-09-14 09:52:08
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /20yyq/networkmanager/route.go
// @@
package networkmanager

import (
	"net"
	"syscall"
	"encoding/binary"

	"github.com/20yyq/packet"
	"github.com/20yyq/netlink"
)

type Routes struct {
	*packet.RtMsg
	iifIdx 		int
	oifIdx 		int
	Priority 	uint32
	Dst 		net.IP
	Src 		net.IP
	Gw 			net.IP
}

func (ifi *Interface) RouteList() ([]*Routes, error) {
	var res []*Routes
	var err error
	count, wait := 0, false
	sm := netlink.SendNLMessage{
		NlMsghdr: &packet.NlMsghdr{Type: RTM_GETROUTE, Flags: syscall.NLM_F_REQUEST|NLM_F_DUMP, Seq: ifi.req},
		Data: (&packet.IfInfomsg{Family: AF_UNSPEC, Index: int32(ifi.iface.Index)}).WireFormat(),
	}
	sm.Len = packet.SizeofNlMsghdr + uint32(len(sm.Data))
	rm := netlink.ReceiveNLMessage{Data: make([]byte, 1024)}
Loop:
	err = ifi.conn.Exchange(&sm, &rm)
	if err == nil {
		count++
		if res, err = ifi.deserializeRtMsgMessages(&rm); res == nil && err == nil {
			wait = true
		}
		if wait && 3 > count {
			goto Loop
		}
	}
	return res, err

}

func (ifi *Interface) AddRoute(r Routes) error {
	sm := netlink.SendNLMessage{
		NlMsghdr: &packet.NlMsghdr{Type: RTM_NEWROUTE, Flags: syscall.NLM_F_REQUEST|NLM_F_CREATE|NLM_F_EXCL|NLM_F_ACK, Seq: ifi.req},
		Data: SerializeRoutes(&r, uint32(ifi.iface.Index)),
	}
	sm.Len = packet.SizeofNlMsghdr + uint32(len(sm.Data))
	rm := netlink.ReceiveNLMessage{Data: make([]byte, 128)}
	err := ifi.conn.Exchange(&sm, &rm)
	if err == nil {
		err = DeserializeNlMsgerr(rm.MsgList[0])
	}
	return err
}

func (ifi *Interface) RemoveRoute(r Routes) error {
	sm := netlink.SendNLMessage{
		NlMsghdr: &packet.NlMsghdr{Type: RTM_DELROUTE, Flags: syscall.NLM_F_REQUEST|NLM_F_ACK, Seq: ifi.req},
		Data: SerializeRoutes(&r, uint32(ifi.iface.Index)),
	}
	sm.Len = packet.SizeofNlMsghdr + uint32(len(sm.Data))
	rm := netlink.ReceiveNLMessage{Data: make([]byte, 128)}
	err := ifi.conn.Exchange(&sm, &rm)
	if err == nil {
		err = DeserializeNlMsgerr(rm.MsgList[0])
	}
	return err
}

func (ifi *Interface) ReplaceRoute(r *Routes) error {
	sm := netlink.SendNLMessage{
		NlMsghdr: &packet.NlMsghdr{Type: RTM_NEWROUTE, Flags: syscall.NLM_F_REQUEST|NLM_F_ACK|NLM_F_REPLACE, Seq: ifi.req},
		Data: SerializeRoutes(r, uint32(ifi.iface.Index)),
	}
	sm.Len = packet.SizeofNlMsghdr + uint32(len(sm.Data))
	rm := netlink.ReceiveNLMessage{Data: make([]byte, 128)}
	err := ifi.conn.Exchange(&sm, &rm)
	if err == nil {
		err = DeserializeNlMsgerr(rm.MsgList[0])
	}
	return err
}

func SerializeRoutes(r *Routes, idx uint32) []byte {
	if r.RtMsg == nil {
		r.RtMsg = &packet.RtMsg{}
	}
	r.Family = AF_INET
	data := make([]byte, packet.SizeofRtMsg)
	data = appendSliceByte(data, RTA_OIF, binary.LittleEndian.AppendUint32(nil, idx))
	if r.Priority > 0 {
		data = appendSliceByte(data, RTA_PRIORITY, binary.LittleEndian.AppendUint32(nil, r.Priority))
	}
	if r.Gw != nil {
		b := r.Gw.To4()
		if len(b) == 0 {
			b = r.Gw.To16()
			r.Family = AF_INET6
		}
		r.Table, r.Protocol, r.Type, r.Scope = RT_TABLE_MAIN, RTPROT_BOOT, RTN_UNICAST, RT_SCOPE_UNIVERSE
		data = appendSliceByte(data, RTA_GATEWAY, b)
	} else {
		if r.Dst != nil {
			b := r.Dst.To4()
			if len(b) == 0 {
				b = r.Dst.To16()
				r.Family = AF_INET6
			}
			data = appendSliceByte(data, RTA_DST, b)
			dstlen, _ := r.Dst.DefaultMask().Size()
			r.Dst_len = uint8(dstlen)
		}
		if r.Src != nil {
			b := r.Src.To4()
			if len(b) == 0 {
				b = r.Src.To16()
				r.Family = AF_INET6
			}
			data = appendSliceByte(data, RTA_PREFSRC, b)
			srclen, _ := r.Src.DefaultMask().Size()
			r.Src_len = uint8(srclen)
		}
	}
	res := r.WireFormat()
	copy(data[:packet.SizeofRtMsg], res)
	return data
}
