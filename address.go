// @@
// @ Author       : Eacher
// @ Date         : 2023-06-27 09:39:36
// @ LastEditTime : 2023-06-27 11:58:28
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /networkmanager/address.go
// @@
package networkmanager

import (
	"fmt"
	"net"
	"time"
	"syscall"
)

type Addrs struct {
	label 		string
	scope 		uint8
	flags 		uint8
	Local 		net.IP
	Broadcast 	net.IP
	Anycast 	net.IP
	Netmask 	net.IP

	Cache 		*cacheInfo
}

func (ifi *Interface) IPList() ([]*Addrs, error) {
	var res []*Addrs
	var nl *NetlinkMessage
	var err error
	count := 0
Loop:
	nl, err = ifi.request(syscall.RTM_GETADDR, syscall.NLM_F_DUMP, 
		IfInfomsgToSliceByte(syscall.AF_UNSPEC, 0x00, 0x00, int32(ifi.iface.Index)))
	if err == nil {
		count++
		nl.mutex.Lock()
		if nl.wait {
			nl.cond.Wait()
		}
		nl.wait = false
		nl.mutex.Unlock()
		time.Sleep(time.Millisecond*50)
		if res, err = nl.deserializeIfAddrmsgMessages(ifi.iface); res == nil && err == nil {
			nl.wait = true
		}
		go ifi.deleteTimeOverNotify(nl)
		if nl.wait && 3 > count {
			goto Loop
		}
	}
	return res, err
}

func (ifi *Interface) AddIP(a Addrs) error {
	data := SerializeAddrs(&a, uint32(ifi.iface.Index))
	nl, err := ifi.request(syscall.RTM_NEWADDR, syscall.NLM_F_REQUEST|syscall.NLM_F_EXCL|syscall.NLM_F_ACK, data)
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

func (ifi *Interface) RemoveIP(a Addrs) error {
	data := SerializeAddrs(&a, uint32(ifi.iface.Index))
	nl, err := ifi.request(syscall.RTM_DELADDR, syscall.NLM_F_ACK, data)
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
func (ifi *Interface) ReplaceIP(a *Addrs) error {
	if a.Cache == nil {
		return fmt.Errorf("34")
	}
	data := SerializeAddrs(a, uint32(ifi.iface.Index))
	nl, err := ifi.request(syscall.RTM_NEWADDR, syscall.NLM_F_REQUEST|syscall.NLM_F_REPLACE|syscall.NLM_F_ACK, data)
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

func SerializeAddrs(a *Addrs, idx uint32) []byte {
	family := syscall.AF_INET
	if len(a.Local) == net.IPv6len {
		family = syscall.AF_INET6
	}
	mask := a.Local.DefaultMask()
	prefixlen, masklen := mask.Size()
	res := IfAddrmsgToSliceByte(uint8(family), uint8(prefixlen), a.flags, a.scope, idx)
	localAddr := a.Local.To4()
	if family == syscall.AF_INET6 {
		localAddr = a.Local.To16()
	}
	data := make([]byte, len(res))
	copy(data, res)
	tmp := data
	res = RtAttrToSliceByte(syscall.IFA_LOCAL, localAddr)
	data = make([]byte, len(tmp) + len(res))
	copy(data[:len(tmp)], tmp)
	copy(data[len(tmp):], res)
	tmp = data
	if a.Netmask != nil {
		res = RtAttrToSliceByte(syscall.IFA_ADDRESS, a.Netmask)
		data = make([]byte, len(tmp) + len(res))
		copy(data[:len(tmp)], tmp)
		copy(data[len(tmp):], res)
		tmp = data
	}
	if a.Broadcast == nil && prefixlen < 31 {
		broadcast := make(net.IP, masklen/8)
		for i := range localAddr {
			broadcast[i] = localAddr[i] | ^mask[i]
		}
		a.Broadcast = broadcast
	}
	if a.Broadcast != nil {
		res = RtAttrToSliceByte(syscall.IFA_BROADCAST, a.Broadcast)
		data = make([]byte, len(tmp) + len(res))
		copy(data[:len(tmp)], tmp)
		copy(data[len(tmp):], res)
		tmp = data
	}
	if a.label != "" {
		res = RtAttrToSliceByte(syscall.IFA_LABEL, []byte(a.label))
		data = make([]byte, len(tmp) + len(res))
		copy(data[:len(tmp)], tmp)
		copy(data[len(tmp):], res)
		tmp = data
	}
	if a.Cache != nil && (a.Cache.PreferredLft > 0 || a.Cache.ValidLft > 0) {
		tmpByte := (*(*[16]byte)(unsafe.Pointer(a.Cache)))[:]
		res = RtAttrToSliceByte(syscall.IFA_CACHEINFO, tmpByte)
		data = make([]byte, len(tmp) + len(res))
		copy(data[:len(tmp)], tmp)
		copy(data[len(tmp):], res)
		tmp = data
	}
	return data
}
