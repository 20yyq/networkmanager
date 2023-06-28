// @@
// @ Author       : Eacher
// @ Date         : 2023-06-27 09:39:36
// @ LastEditTime : 2023-06-28 17:02:08
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
	netmask 	net.IP
	Local 		net.IP
	Broadcast 	net.IP
	Anycast 	net.IP

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

func (ifi *Interface) ReplaceIP(a *Addrs) error {
	if a.netmask != nil {
		if err := ifi.ReplaceIP(&Addrs{Local: a.netmask, Cache: &CacheInfo{PreferredLft: 1, ValidLft: 1}}); err != nil {
			return err
		}
	}
	data := SerializeAddrs(a, uint32(ifi.iface.Index))
	nl, err := ifi.request(syscall.RTM_NEWADDR, syscall.NLM_F_REQUEST|syscall.NLM_F_ACK|syscall.NLM_F_REPLACE, data)
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
	localAddr := a.Local.To4()
	if len(localAddr) == 0 {
		family = syscall.AF_INET6
		localAddr = a.Local.To16()
	}
	mask := a.Local.DefaultMask()
	prefixlen, masklen := mask.Size()
	data := IfAddrmsgToSliceByte(uint8(family), uint8(prefixlen), a.flags, a.scope, idx)
	data = appendSliceByte(data, syscall.IFA_LOCAL, localAddr)
	if a.Broadcast == nil && prefixlen < 31 {
		broadcast := make([]byte, masklen/8)
		for i := range localAddr {
			broadcast[i] = localAddr[i] | ^mask[i]
		}
		a.Broadcast = net.IPv4(broadcast[0], broadcast[1], broadcast[2], broadcast[3])
	}
	if a.Broadcast != nil {
		data = appendSliceByte(data, syscall.IFA_BROADCAST, a.Broadcast.To4())
	}
	if a.Anycast != nil {
		data = appendSliceByte(data, syscall.IFA_ANYCAST, a.Anycast.To4())
	}
	if a.label != "" {
		data = appendSliceByte(data, syscall.IFA_LABEL, []byte(a.label))
	}
	if a.Cache != nil && (a.Cache.PreferredLft > 0 || a.Cache.ValidLft > 0) {
		data = appendSliceByte(data, syscall.IFA_CACHEINFO, (*(*[16]byte)(unsafe.Pointer(a.Cache)))[:])
	}
	return data
}
