// @@
// @ Author       : Eacher
// @ Date         : 2023-06-27 09:38:13
// @ LastEditTime : 2023-06-28 15:26:34
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
	*syscall.RtMsg
	iifIdx 		int
	oifIdx 		int
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

func (ifi *Interface) ReplaceRoute(r *Routes) error {
	data := SerializeRoutes(r, uint32(ifi.iface.Index))
	nl, err := ifi.request(syscall.RTM_NEWROUTE, syscall.NLM_F_REQUEST|syscall.NLM_F_ACK|syscall.NLM_F_REPLACE, data)
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
	var (
		family uint8 = syscall.AF_INET
		flags uint32
		dstlen, srclen int
		data []byte
	)
	data = make([]byte, syscall.SizeofRtMsg)
	data = appendSliceByte(data, syscall.RTA_OIF, binary.LittleEndian.AppendUint32(nil, idx))
	if r.Priority > 0 {
		data = appendSliceByte(data, syscall.RTA_PRIORITY, binary.LittleEndian.AppendUint32(nil, r.Priority))
	}
	if r.Gw != nil {
		b := r.Gw.To4()
		if len(b) == 0 {
			b = r.Gw.To16()
			family = syscall.AF_INET6
		}
		data = appendSliceByte(data, syscall.RTA_GATEWAY, b)
	} else {
		if r.Dst != nil {
			b := r.Dst.To4()
			if len(b) == 0 {
				b = r.Dst.To16()
				family = syscall.AF_INET6
			}
			data = appendSliceByte(data, syscall.RTA_DST, b)
			dstlen, _ = r.Dst.DefaultMask().Size()
		}
		if r.Src != nil {
			b := r.Src.To4()
			if len(b) == 0 {
				b = r.Src.To16()
				family = syscall.AF_INET6
			}
			data = appendSliceByte(data, syscall.RTA_PREFSRC, b)
			srclen, _ = r.Src.DefaultMask().Size()
		}
	}
	res := RtMsgToSliceByte(family, uint8(dstlen), uint8(srclen), r.RtMsg.Tos, r.RtMsg.Table, r.RtMsg.Protocol, r.RtMsg.Scope, r.RtMsg.Type, flags)
	copy(data[:syscall.SizeofRtMsg], res)
	return data
}
