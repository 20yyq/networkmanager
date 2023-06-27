// @@
// @ Author       : Eacher
// @ Date         : 2023-06-27 09:38:13
// @ LastEditTime : 2023-06-27 11:58:23
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /networkmanager/route.go
// @@
package networkmanager

import (
	"fmt"
	"net"
	"time"
	"syscall"
)

type Routes struct {
	iifIdx 		int
	oifIdx 		int
	Scope 		uint8
	Table 		uint32
	Priority 	uint32
	Dst 		net.IP
	Src 		net.IP
	Gw 			net.IP
}

func (ifi *Interface) RouteList() ([]*Routes, error) {
	var res []*Routes
	var nl *NetlinkMessage
	var err error
	count := 0
Loop:
	nl, err = ifi.request(syscall.RTM_GETROUTE, syscall.NLM_F_DUMP, 
		IfInfomsgToSliceByte(syscall.AF_UNSPEC, 0x00, 0x00, int32(ifi.iface.Index)))
	if err == nil {
		count++
		nl.mutex.Lock()
		if nl.wait {
			nl.cond.Wait()
		}
		nl.wait = false
		nl.mutex.Unlock()
		time.Sleep(time.Millisecond*100)
		if res, err = nl.deserializeRtMsgMessages(ifi.iface); res == nil && err == nil {
			nl.wait = true
		}
		go ifi.deleteTimeOverNotify(nl)
		if nl.wait && 3 > count {
			goto Loop
		}
	}
	return res, err

}

func (ifi *Interface) AddRoute(r Routes) error {
	data := SerializeRoutes(&r, uint32(ifi.iface.Index))
	nl, err := ifi.request(syscall.RTM_NEWROUTE, syscall.NLM_F_REQUEST|syscall.NLM_F_EXCL|syscall.NLM_F_ACK, data)
	if err == nil {
		nl.mutex.Lock()
		if nl.wait {
			nl.cond.Wait()
		}
		nl.wait = false
		nl.mutex.Unlock()
		go ifi.deleteTimeOverNotify(nl)
		err = DeserializeNlMsgerr(nl.Message[0])
	}
	return err
}

func (ifi *Interface) RemoveRoute(r Routes) error {
	data := SerializeRoutes(&r, uint32(ifi.iface.Index))
	nl, err := ifi.request(syscall.RTM_DELROUTE, syscall.NLM_F_ACK, data)
	if err == nil {
		nl.mutex.Lock()
		if nl.wait {
			nl.cond.Wait()
		}
		nl.wait = false
		nl.mutex.Unlock()
		go ifi.deleteTimeOverNotify(nl)
		err = DeserializeNlMsgerr(nl.Message[0])
	}
	return err
}

// TODO 暂时不可用
func (ifi *Interface) ReplaceRoute(r *Routes) error {
	data := SerializeRoutes(r, uint32(ifi.iface.Index))
	nl, err := ifi.request(syscall.RTM_NEWROUTE, syscall.NLM_F_REQUEST|syscall.NLM_F_REPLACE|syscall.NLM_F_ACK, data)
	if err == nil {
		nl.mutex.Lock()
		if nl.wait {
			nl.cond.Wait()
		}
		nl.wait = false
		nl.mutex.Unlock()
		go ifi.deleteTimeOverNotify(nl)
		err = DeserializeNlMsgerr(nl.Message[0])
	}
	return err
}

func SerializeRoutes(r *Routes, idx uint32) []byte {
	dstlen, srclen := uint8(len(r.Dst)), uint8(len(r.Src))
	if dstlen != srclen {
		return nil
	}
	var (
		protocol uint8 	= syscall.RTPROT_BOOT
		family uint8 	= syscall.AF_INET
		table uint8 	= syscall.RT_TABLE_MAIN
		tos uint8 		= 0
		types uint8 	= syscall.RTN_UNICAST
		flags uint32
	)
	tmp := make([]byte, syscall.SizeofRtMsg)
	res := RtAttrToSliceByte(syscall.RTA_OIF, binary.LittleEndian.AppendUint32(nil, idx))
	data := make([]byte, len(tmp) + len(res))
	copy(data[:len(tmp)], tmp)
	copy(data[len(tmp):], res)
	tmp = data
	if r.Priority > 0 {
		res = RtAttrToSliceByte(syscall.RTA_PRIORITY, binary.LittleEndian.AppendUint32(nil, r.Priority))
		data = make([]byte, len(tmp) + len(res))
		copy(data[:len(tmp)], tmp)
		copy(data[len(tmp):], res)
		tmp = data
	}
	table = uint8(r.Table)
	if r.Table > 254 {
		res = RtAttrToSliceByte(syscall.RTA_TABLE, binary.LittleEndian.AppendUint32(nil, r.Table))
		data = make([]byte, len(tmp) + len(res))
		copy(data[:len(tmp)], tmp)
		copy(data[len(tmp):], res)
		tmp, table = data, syscall.RT_TABLE_UNSPEC
	}
	if r.Gw != nil {
		res = RtAttrToSliceByte(syscall.RTA_GATEWAY, r.Gw)
		data = make([]byte, len(tmp) + len(res))
		copy(data[:len(tmp)], tmp)
		copy(data[len(tmp):], res)
		tmp = data
		if len(r.Gw) == net.IPv6len {
			family = syscall.AF_INET6
		}
		res = RtMsgToSliceByte(family, dstlen, srclen, tos, table, protocol, r.Scope, types, flags)
		copy(data[:syscall.SizeofRtMsg], res)
		return data
	}
	res = RtAttrToSliceByte(syscall.RTA_DST, r.Dst)
	data = make([]byte, len(tmp) + len(res))
	copy(data[:len(tmp)], tmp)
	copy(data[len(tmp):], res)
	tmp = data
	res = RtAttrToSliceByte(syscall.RTA_SRC, r.Src)
	data = make([]byte, len(tmp) + len(res))
	copy(data[:len(tmp)], tmp)
	copy(data[len(tmp):], res)
	tmp = data
	if len(r.Dst) == net.IPv6len {
		family = syscall.AF_INET6
	}
	res = RtMsgToSliceByte(family, dstlen, srclen, tos, table, protocol, r.Scope, types, flags)
	copy(data[:syscall.SizeofRtMsg], res)
	return data
}
