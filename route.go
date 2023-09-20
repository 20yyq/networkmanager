// @@
// @ Author       : Eacher
// @ Date         : 2023-06-27 09:38:13
// @ LastEditTime : 2023-09-20 11:33:14
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
	var err error
	sm := netlink.SendNLMessage{
		NlMsghdr: &packet.NlMsghdr{Type: RTM_GETROUTE, Flags: NLM_F_REQUEST|NLM_F_DUMP, Seq: randReq()},
	}
	sm.Attrs = append(sm.Attrs, packet.IfInfomsg{Family: AF_UNSPEC, Index: int32(ifi.iface.Index)})
	rm := netlink.ReceiveNLMessage{Data: make([]byte, 1024)}
	if err = ifi.conn.Exchange(&sm, &rm); err == nil {
		return ifi.deserializeRtMsgMessages(&rm)
	}
	return nil, err

}

func (ifi *Interface) AddRoute(r Routes) error {
	sm := netlink.SendNLMessage{
		NlMsghdr: &packet.NlMsghdr{Type: RTM_NEWROUTE, Flags: NLM_F_REQUEST|NLM_F_CREATE|NLM_F_EXCL|NLM_F_ACK, Seq: randReq()},
	}
	sm.Attrs = append(sm.Attrs, SerializeRoutes(&r, uint32(ifi.iface.Index))...)
	rm := netlink.ReceiveNLMessage{Data: make([]byte, 1024)}
	err := ifi.conn.Exchange(&sm, &rm)
	if err == nil {
		if rm.MsgList[0].Header.Type != RTM_NEWROUTE {
			err = DeserializeNlMsgerr(rm.MsgList[0])
		}
	}
	return err
}

func (ifi *Interface) RemoveRoute(r Routes) error {
	sm := netlink.SendNLMessage{
		NlMsghdr: &packet.NlMsghdr{Type: RTM_DELROUTE, Flags: NLM_F_REQUEST|NLM_F_ACK, Seq: randReq()},
	}
	sm.Attrs = append(sm.Attrs, SerializeRoutes(&r, uint32(ifi.iface.Index))...)
	rm := netlink.ReceiveNLMessage{Data: make([]byte, 1024)}
	err := ifi.conn.Exchange(&sm, &rm)
	if err == nil {
		if rm.MsgList[0].Header.Type != RTM_DELROUTE {
			err = DeserializeNlMsgerr(rm.MsgList[0])
		}
	}
	return err
}

func (ifi *Interface) ReplaceRoute(r *Routes) error {
	sm := netlink.SendNLMessage{
		NlMsghdr: &packet.NlMsghdr{Type: RTM_NEWROUTE, Flags: NLM_F_REQUEST|NLM_F_ACK|NLM_F_REPLACE, Seq: randReq()},
	}
	sm.Attrs = append(sm.Attrs, SerializeRoutes(r, uint32(ifi.iface.Index))...)
	rm := netlink.ReceiveNLMessage{Data: make([]byte, 1024)}
	err := ifi.conn.Exchange(&sm, &rm)
	if err == nil {
		if rm.MsgList[0].Header.Type != RTM_NEWROUTE {
			err = DeserializeNlMsgerr(rm.MsgList[0])
		}
	}
	return err
}

func SerializeRoutes(r *Routes, idx uint32) []packet.Attrs {
	attrs := make([]packet.Attrs, 1)
	if r.RtMsg == nil {
		r.RtMsg = &packet.RtMsg{}
	}
	r.Family = AF_INET
	data := binary.LittleEndian.AppendUint32(nil, idx)
	attrs = append(attrs, packet.RtAttr{&syscall.RtAttr{uint16(len(data) + packet.SizeofRtAttr), RTA_OIF}, data})
	if r.Priority > 0 {
		data = binary.LittleEndian.AppendUint32(nil, r.Priority)
		attrs = append(attrs, packet.RtAttr{&syscall.RtAttr{uint16(len(data) + packet.SizeofRtAttr), RTA_PRIORITY}, data})
	}
	if r.Gw != nil {
		b := r.Gw.To4()
		if len(b) == 0 {
			b = r.Gw.To16()
			r.Family = AF_INET6
		}
		r.Table, r.Protocol, r.Type, r.Scope = RT_TABLE_MAIN, RTPROT_BOOT, RTN_UNICAST, RT_SCOPE_UNIVERSE
		attrs = append(attrs, packet.RtAttr{&syscall.RtAttr{uint16(len(b) + packet.SizeofRtAttr), RTA_GATEWAY}, b})
	} else {
		if r.Dst != nil {
			b := r.Dst.To4()
			if len(b) == 0 {
				b = r.Dst.To16()
				r.Family = AF_INET6
			}
			attrs = append(attrs, packet.RtAttr{&syscall.RtAttr{uint16(len(b) + packet.SizeofRtAttr), RTA_DST}, b})
			dstlen, _ := r.Dst.DefaultMask().Size()
			r.Dst_len = uint8(dstlen)
		}
		if r.Src != nil {
			b := r.Src.To4()
			if len(b) == 0 {
				b = r.Src.To16()
				r.Family = AF_INET6
			}
			attrs = append(attrs, packet.RtAttr{&syscall.RtAttr{uint16(len(b) + packet.SizeofRtAttr), RTA_PREFSRC}, b})
			srclen, _ := r.Src.DefaultMask().Size()
			r.Src_len = uint8(srclen)
		}
	}
	attrs[0] = packet.Attrs(r.RtMsg)
	return attrs
}
