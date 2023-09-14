// @@
// @ Author       : Eacher
// @ Date         : 2023-06-26 08:01:05
// @ LastEditTime : 2023-09-14 09:51:13
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /20yyq/networkmanager/netlinkmessage.go
// @@
package networkmanager

import (
	"fmt"
	"net"
	"unsafe"
	"syscall"
	"encoding/binary"

	"github.com/20yyq/packet"
	"github.com/20yyq/netlink"
)

//go:linkname rtaAlignOf syscall.rtaAlignOf
func rtaAlignOf(attrlen int) int

func DeserializeNlMsgerr(nlm *packet.NetlinkMessage) error {
	if len(nlm.Data) < packet.SizeofNlMsgerr {
		return syscall.Errno(34)
	}
	msg := packet.NewNlMsgerr(([packet.SizeofNlMsgerr]byte)(nlm.Data[:packet.SizeofNlMsgerr]))
	if msg.Error < 0 {
		msg.Error *= -1
	}
	if msg.Error > 0 {
		return syscall.Errno(msg.Error)
	}
	return nil
}

func (ifi *Interface) deserializeIfAddrmsgMessages(rm *netlink.ReceiveNLMessage) ([]*Addrs, error) {
	var res []*Addrs
	name := ifi.iface.Name + string([]byte{0})
	for _, m := range rm.MsgList {
		if l, err := ParseNetlinkRouteAttr(m); err == nil {
			single := Addrs{IfAddrmsg: packet.NewIfAddrmsg(([packet.SizeofIfAddrmsg]byte)(m.Data[:packet.SizeofIfAddrmsg])), label: ""}
			for _, v := range l {
				switch v.Attr.Type {
				case IFA_ADDRESS:
					single.address = net.IPv4(v.Value[0], v.Value[1], v.Value[2], v.Value[3])
				case IFA_LOCAL:
					single.Local = net.IPv4(v.Value[0], v.Value[1], v.Value[2], v.Value[3])
				case IFA_BROADCAST:
					single.Broadcast = net.IPv4(v.Value[0], v.Value[1], v.Value[2], v.Value[3])
				case IFA_ANYCAST:
					single.Anycast = net.IPv4(v.Value[0], v.Value[1], v.Value[2], v.Value[3])
				case IFA_LABEL:
					single.label = string(v.Value)
				case IFA_CACHEINFO:
					single.Cache = (*cacheInfo)(unsafe.Pointer(&v.Value[:16][0]))
				default:
					fmt.Println("IFLA_COST", v.Attr.Type, v.Value, single)
				}
			}
			if single.label == name {
				res = append(res, &single)
			}
			continue
		}
		if 1 > len(res) {
			return nil, DeserializeNlMsgerr(m)
		}
	}
	return res, nil
}

func (ifi *Interface) deserializeRtMsgMessages(rm *netlink.ReceiveNLMessage) ([]*Routes, error) {
	var res []*Routes
	for _, m := range rm.MsgList {
		if l, err := ParseNetlinkRouteAttr(m); err == nil {
			single := Routes{
				RtMsg: packet.NewRtMsg(([packet.SizeofRtMsg]byte)(m.Data[:packet.SizeofRtMsg])),
				oifIdx: -9999, iifIdx: -9999,
			}
			for _, v := range l {
				switch v.Attr.Type {
				case RTA_DST: // 目标地址
					single.Dst = net.IP(v.Value)
				case RTA_SRC: // 源地址
					
				case RTA_PREFSRC:
					single.Src = net.IP(v.Value)
				case RTA_IIF: // 输入接口iifIdx
					single.iifIdx = int(binary.LittleEndian.Uint32(v.Value))
				case RTA_OIF: // 输出接口oifIdx
					single.oifIdx = int(binary.LittleEndian.Uint32(v.Value))
				case RTA_GATEWAY:
					single.Gw = net.IP(v.Value)
				case RTA_PRIORITY:
					single.Priority = binary.LittleEndian.Uint32(v.Value)
				case RTA_METRICS:

				case RTA_FLOW: // 所属领域

				case RTA_TABLE:
					
				case RTA_CACHEINFO:
					fmt.Println("syscall.RTA_CACHEINFO", v.Value)
				case RTNLGRP_ND_USEROPT:
					fmt.Println("syscall.RTNLGRP_ND_USEROPT", v.Value)
				default:
					fmt.Println("RTA_MULTIPATH", v.Attr.Type, v.Value, single)
				}
			}
			if single.oifIdx == ifi.iface.Index {
				res = append(res, &single)
			}
			continue
		}
		if 1 > len(res) {
			return nil, DeserializeNlMsgerr(m)
		}
	}
	return res, nil
}

// ParseNetlinkRouteAttr parses m's payload as an array of netlink
// route attributes and returns the slice containing the
// NetlinkRouteAttr structures.
func ParseNetlinkRouteAttr(m *packet.NetlinkMessage) ([]syscall.NetlinkRouteAttr, error) {
	var b []byte
	switch m.Header.Type {
	case syscall.RTM_NEWLINK, syscall.RTM_DELLINK:
		b = m.Data[syscall.SizeofIfInfomsg:]
	case syscall.RTM_NEWADDR, syscall.RTM_DELADDR:
		b = m.Data[syscall.SizeofIfAddrmsg:]
	case syscall.RTM_NEWROUTE, syscall.RTM_DELROUTE:
		b = m.Data[syscall.SizeofRtMsg:]
	default:
		return nil, syscall.EINVAL
	}
	var attrs []syscall.NetlinkRouteAttr
	for len(b) >= syscall.SizeofRtAttr {
		a, vbuf, alen, err := netlinkRouteAttrAndValue(b)
		if err != nil {
			return nil, err
		}
		ra := syscall.NetlinkRouteAttr{Attr: *a, Value: vbuf[:int(a.Len)-syscall.SizeofRtAttr]}
		attrs = append(attrs, ra)
		b = b[alen:]
	}
	return attrs, nil
}

func netlinkRouteAttrAndValue(b []byte) (*syscall.RtAttr, []byte, int, error) {
	a := (*syscall.RtAttr)(unsafe.Pointer(&b[0]))
	if int(a.Len) < syscall.SizeofRtAttr || int(a.Len) > len(b) {
		return nil, nil, 0, syscall.EINVAL
	}
	return a, b[syscall.SizeofRtAttr:], rtaAlignOf(int(a.Len)), nil
}
