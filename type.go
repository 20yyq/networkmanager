// @@
// @ Author       : Eacher
// @ Date         : 2023-06-29 08:26:37
// @ LastEditTime : 2023-06-29 11:12:44
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /networkmanager/type.go
// @@
package networkmanager

import (
	"unsafe"
	"syscall"
)

type IfInfomsg syscall.IfInfomsg
type IfAddrmsg syscall.IfAddrmsg
type RtMsg syscall.RtMsg
type NlMsghdr syscall.NlMsghdr
type NlMsgerr struct {
	Error int32
	Msg   NlMsghdr
}

func NewIfInfomsg(b []byte) (info *IfInfomsg) {
	if !(len(b) < SizeofIfInfomsg) {
		info = (*IfInfomsg)(unsafe.Pointer(&b[0]))
	}
	return
}

func (info *IfInfomsg) WireFormat() []byte {
	b := make([]byte, SizeofIfInfomsg)
	b[0], b[1] = byte(info.Family), byte(info.X__ifi_pad)
	*(*uint16)(unsafe.Pointer(&b[2:4][0])) = info.Type
	*(*int32)(unsafe.Pointer(&b[4:8][0])) = info.Index
	*(*uint32)(unsafe.Pointer(&b[8:12][0])) = info.Flags
	*(*uint32)(unsafe.Pointer(&b[12:16][0])) = info.Change
	return b
}

func NewIfAddrmsg(b []byte) (addr *IfAddrmsg) {
	if !(len(b) < SizeofIfAddrmsg) {
		addr = (*IfAddrmsg)(unsafe.Pointer(&b[0]))
	}
	return
}

func (addr *IfAddrmsg) WireFormat() []byte {
	b := make([]byte, SizeofIfAddrmsg)
	b[0], b[1], b[2], b[3] = byte(addr.Family), byte(addr.Prefixlen), byte(addr.Flags), byte(addr.Scope)
	*(*uint32)(unsafe.Pointer(&b[4:8][0])) = addr.Index
	return b
}

func NewRtMsg(b []byte) (rtmsg *RtMsg) {
	if !(len(b) < SizeofRtMsg) {
		rtmsg = (*RtMsg)(unsafe.Pointer(&b[0]))
	}
	return
}

func (rtmsg *RtMsg) WireFormat() []byte {
	b := make([]byte, SizeofRtMsg)
	b[0], b[1], b[2], b[3] = byte(rtmsg.Family), byte(rtmsg.Dst_len), byte(rtmsg.Src_len), byte(rtmsg.Tos)
	b[4], b[5], b[6], b[7] = byte(rtmsg.Table), byte(rtmsg.Protocol), byte(rtmsg.Scope), byte(rtmsg.Type)
	*(*uint32)(unsafe.Pointer(&b[8:12][0])) = rtmsg.Flags
	return b
}

func NewNlMsghdr(b []byte) (hdr *NlMsghdr) {
	if !(len(b) < SizeofNlMsghdr) {
		hdr = (*NlMsghdr)(unsafe.Pointer(&b[0]))
	}
	return
}

func (hdr *NlMsghdr) WireFormat() []byte {
	b := make([]byte, SizeofNlMsghdr)
	hdr.wireFormat(b)
	return b
}

func (hdr *NlMsghdr) wireFormat(b []byte) {
	*(*uint32)(unsafe.Pointer(&b[0:4][0])) = hdr.Len
	*(*uint16)(unsafe.Pointer(&b[4:6][0])) = hdr.Type
	*(*uint16)(unsafe.Pointer(&b[6:8][0])) = hdr.Flags
	*(*uint32)(unsafe.Pointer(&b[8:12][0])) = hdr.Seq
	*(*uint32)(unsafe.Pointer(&b[12:16][0])) = hdr.Pid
}

func NewNlMsgerr(b []byte) (nlmsg *NlMsgerr) {
	if !(len(b) < SizeofNlMsgerr) {
		nlmsg = (*NlMsgerr)(unsafe.Pointer(&b[0]))
	}
	return
}

func (nlmsg *NlMsgerr) WireFormat() []byte {
	b := make([]byte, SizeofNlMsgerr)
	*(*int32)(unsafe.Pointer(&b[0:4][0])) = nlmsg.Error
	nlmsg.Msg.wireFormat(b[4:])
	return b
}

const (
	SOCK_CLOEXEC 	= syscall.SOCK_CLOEXEC
	SOL_SOCKET 		= syscall.SOL_SOCKET
	SOCK_RAW 		= syscall.SOCK_RAW
	SO_RCVTIMEO 	= syscall.SO_RCVTIMEO
	NLMSG_HDRLEN 	= syscall.NLMSG_HDRLEN
	NETLINK_ROUTE 	= syscall.NETLINK_ROUTE

	AF_NETLINK 		= syscall.AF_NETLINK
	AF_UNSPEC 		= syscall.AF_UNSPEC
	AF_INET 		= syscall.AF_INET
	AF_INET6 		= syscall.AF_INET6

	IFA_LOCAL 		= syscall.IFA_LOCAL
	IFA_BROADCAST 	= syscall.IFA_BROADCAST
	IFA_ANYCAST 	= syscall.IFA_ANYCAST
	IFA_LABEL 		= syscall.IFA_LABEL
	IFA_CACHEINFO 	= syscall.IFA_CACHEINFO
	IFA_ADDRESS 	= syscall.IFA_ADDRESS
	IFF_UP 			= syscall.IFF_UP

	NLM_F_CREATE 	= syscall.NLM_F_CREATE
	NLM_F_REQUEST 	= syscall.NLM_F_REQUEST
	NLM_F_EXCL 		= syscall.NLM_F_EXCL
	NLM_F_ACK 		= syscall.NLM_F_ACK
	NLM_F_DUMP 		= syscall.NLM_F_DUMP
	NLM_F_REPLACE 	= syscall.NLM_F_REPLACE

	RTM_NEWADDR 	= syscall.RTM_NEWADDR
	RTM_GETADDR 	= syscall.RTM_GETADDR
	RTM_DELADDR 	= syscall.RTM_DELADDR
	RTM_NEWLINK 	= syscall.RTM_NEWLINK
	RTM_GETROUTE 	= syscall.RTM_GETROUTE
	RTM_NEWROUTE 	= syscall.RTM_NEWROUTE
	RTM_DELROUTE 	= syscall.RTM_DELROUTE

	RTA_DST 		= syscall.RTA_DST
	RTA_SRC 		= syscall.RTA_SRC
	RTA_PREFSRC 	= syscall.RTA_PREFSRC
	RTA_IIF 		= syscall.RTA_IIF
	RTA_OIF 		= syscall.RTA_OIF
	RTA_GATEWAY 	= syscall.RTA_GATEWAY
	RTA_PRIORITY 	= syscall.RTA_PRIORITY
	RTA_METRICS 	= syscall.RTA_METRICS
	RTA_FLOW 		= syscall.RTA_FLOW
	RTA_TABLE 		= syscall.RTA_TABLE
	RTA_CACHEINFO 	= syscall.RTA_CACHEINFO

	RT_TABLE_MAIN 	= syscall.RT_TABLE_MAIN
	RT_SCOPE_UNIVERSE = syscall.RT_SCOPE_UNIVERSE
	RTPROT_BOOT 	= syscall.RTPROT_BOOT
	RTN_UNICAST 	= syscall.RTN_UNICAST

	RTNLGRP_ND_USEROPT = syscall.RTNLGRP_ND_USEROPT

	SizeofNlMsghdr 	= syscall.SizeofNlMsghdr
	SizeofRtAttr 	= syscall.SizeofRtAttr
	SizeofNlMsgerr 	= syscall.SizeofNlMsgerr
	SizeofIfAddrmsg = syscall.SizeofIfAddrmsg
	SizeofRtMsg 	= syscall.SizeofRtMsg
	SizeofIfInfomsg = syscall.SizeofIfInfomsg
)
