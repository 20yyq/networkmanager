// @@
// @ Author       : Eacher
// @ Date         : 2023-06-26 08:01:05
// @ LastEditTime : 2023-09-20 14:03:12
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
	var m *packet.NetlinkMessage
	name, l := ifi.iface.Name + string([]byte{0}), len(rm.MsgList)
	for i := 0; i < l; i++ {
		m = rm.MsgList[i]
		if l, err := packet.ParseNetlinkRouteAttr(m); err == nil {
			single := Addrs{IfAddrmsg: packet.NewIfAddrmsg(([packet.SizeofIfAddrmsg]byte)(m.Data[:packet.SizeofIfAddrmsg])), label: ""}
			for _, v := range l {
				switch v.Type {
				case IFA_ADDRESS:
					single.address = net.IPv4(v.Data[0], v.Data[1], v.Data[2], v.Data[3])
				case IFA_LOCAL:
					single.Local = net.IPv4(v.Data[0], v.Data[1], v.Data[2], v.Data[3])
				case IFA_BROADCAST:
					single.Broadcast = net.IPv4(v.Data[0], v.Data[1], v.Data[2], v.Data[3])
				case IFA_ANYCAST:
					single.Anycast = net.IPv4(v.Data[0], v.Data[1], v.Data[2], v.Data[3])
				case IFA_LABEL:
					single.label = string(v.Data)
				case IFA_CACHEINFO:
					single.Cache = (*cacheInfo)(unsafe.Pointer(&v.Data[:16][0]))
				default:
					fmt.Println("IFLA_COST", v.Type, v.Data, single)
				}
			}
			if single.label == name {
				res = append(res, &single)
			}
			continue
		}
		if i == l - 1 && 0 == len(res) {
			return nil, DeserializeNlMsgerr(m)
		}
	}
	return res, nil
}

func (ifi *Interface) deserializeRtMsgMessages(rm *netlink.ReceiveNLMessage) ([]*Routes, error) {
	var res []*Routes
	var m *packet.NetlinkMessage
	l := len(rm.MsgList)
	for i := 0; i < l; i++ {
		m = rm.MsgList[i]
		if l, err := packet.ParseNetlinkRouteAttr(m); err == nil {
			single := Routes{
				RtMsg: packet.NewRtMsg(([packet.SizeofRtMsg]byte)(m.Data[:packet.SizeofRtMsg])),
				oifIdx: -9999, iifIdx: -9999,
			}
			for _, v := range l {
				switch v.Type {
				case RTA_DST: // 目标地址
					single.Dst = net.IP(v.Data)
				case RTA_SRC: // 源地址
					
				case RTA_PREFSRC:
					single.Src = net.IP(v.Data)
				case RTA_IIF: // 输入接口iifIdx
					single.iifIdx = int(binary.LittleEndian.Uint32(v.Data))
				case RTA_OIF: // 输出接口oifIdx
					single.oifIdx = int(binary.LittleEndian.Uint32(v.Data))
				case RTA_GATEWAY:
					single.Gw = net.IP(v.Data)
				case RTA_PRIORITY:
					single.Priority = binary.LittleEndian.Uint32(v.Data)
				case RTA_METRICS:

				case RTA_FLOW: // 所属领域

				case RTA_TABLE:
					
				case RTA_CACHEINFO:
					fmt.Println("syscall.RTA_CACHEINFO", v.Data)
				case RTNLGRP_ND_USEROPT:
					fmt.Println("syscall.RTNLGRP_ND_USEROPT", v.Data)
				default:
					fmt.Println("RTA_MULTIPATH", v.Type, v.Data, single)
				}
			}
			if single.oifIdx == ifi.iface.Index {
				res = append(res, &single)
			}
			continue
		}
		if i == l - 1 && 0 == len(res) {
			return nil, DeserializeNlMsgerr(m)
		}
	}
	return res, nil
}
