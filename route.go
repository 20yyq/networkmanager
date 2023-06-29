// @@
// @ Author       : Eacher
// @ Date         : 2023-06-27 09:38:13
// @ LastEditTime : 2023-06-29 11:13:38
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /networkmanager/route.go
// @@
package networkmanager

import (
	"log"
	"net"
	"time"
	"encoding/binary"
)

type Routes struct {
	*RtMsg
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
	nl, err = ifi.request(RTM_GETROUTE, NLM_F_DUMP, (&IfInfomsg{Family: AF_UNSPEC, Index: int32(ifi.iface.Index)}).WireFormat())
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
	nl, err := ifi.request(RTM_NEWROUTE, NLM_F_CREATE|NLM_F_EXCL|NLM_F_ACK, data)
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
	nl, err := ifi.request(RTM_DELROUTE, NLM_F_ACK, data)
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
	nl, err := ifi.request(RTM_NEWROUTE, NLM_F_REQUEST|NLM_F_ACK|NLM_F_REPLACE, data)
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
	if r.RtMsg == nil {
		r.RtMsg = &RtMsg{}
	}
	r.Family = AF_INET
	data := make([]byte, SizeofRtMsg)
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
	copy(data[:SizeofRtMsg], res)
	return data
}
