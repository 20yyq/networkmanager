// @@
// @ Author       : Eacher
// @ Date         : 2023-06-26 08:01:05
// @ LastEditTime : 2023-06-29 11:14:20
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /networkmanager/netlinkmessage.go
// @@
package networkmanager

import (
	"fmt"
	"net"
	"sync"
	"unsafe"
	"syscall"
	"encoding/binary"
)

func DeserializeNlMsgerr(nlm *syscall.NetlinkMessage) error {
	if len(nlm.Data) < SizeofNlMsgerr {
		return syscall.Errno(34)
	}
	msg := NewNlMsgerr(nlm.Data[:SizeofNlMsgerr])
	if msg.Error < 0 {
		msg.Error *= -1
	}
	if msg.Error > 0 {
		return syscall.Errno(msg.Error)
	}
	return nil
}

type NetlinkMessage struct {
	req 	uint32
	pid 	uint32
	wait 	bool
	mutex 	sync.Mutex
	cond	*sync.Cond
	
	Message []*syscall.NetlinkMessage
}

type cacheInfo struct {
	PreferredLft 	uint32
	ValidLft 		uint32
	CreadTime 		uint32
	UpdateTime 		uint32
}

func (nlm *NetlinkMessage) deserializeIfAddrmsgMessages(ifi *net.Interface) ([]*Addrs, error) {
	var res []*Addrs
	for _, m := range nlm.Message {
		if l, err := syscall.ParseNetlinkRouteAttr(m); err == nil {
			single := Addrs{IfAddrmsg: NewIfAddrmsg(m.Data[:SizeofIfAddrmsg]), label: ""}
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
			if single.label == ifi.Name {
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

func (nlm *NetlinkMessage) deserializeRtMsgMessages(ifi *net.Interface) ([]*Routes, error) {
	var res []*Routes
	for _, m := range nlm.Message {
		if l, err := syscall.ParseNetlinkRouteAttr(m); err == nil {
			single := Routes{
				RtMsg: NewRtMsg(m.Data[:SizeofRtMsg]),
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
			if single.oifIdx == ifi.Index {
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
