// @@
// @ Author       : Eacher
// @ Date         : 2023-07-01 09:08:50
// @ LastEditTime : 2023-07-04 15:07:45
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /networkmanager/socket/socket.go
// @@
package socket

import (
	"os"
	"time"
	"syscall"
	"sync/atomic"
)

type Socket struct {
	f 		*os.File
	rc 		syscall.RawConn

	closed 	uint32
}

func NewSocket(domain, typ, proto int, name string) (*Socket, error) {
	var conn *Socket
	fd, err := syscall.Socket(domain, typ, proto)
	if err == nil {
		if err = syscall.SetNonblock(fd, true); err == nil {
			conn = &Socket{f: os.NewFile(uintptr(fd), name)}
			if conn.rc, err = conn.f.SyscallConn(); err != nil {
				conn.f.Close()
				conn = nil
			}
		}
	}
	return conn, err
}

func (s *Socket) Close() (err error) {
	if atomic.AddUint32(&s.closed, 1) < 2 {
		err = s.f.Close()
	}
	return 
}

func (s *Socket) Recvfrom(b []byte, flags int) (n int, from syscall.Sockaddr, err error) {
	if atomic.LoadUint32(&s.closed) != 0 {
		return 0, nil, os.NewSyscallError("Recvfrom closed", nil)
	}
	fun := func(fd uintptr) bool {
		if n, from, err = syscall.Recvfrom(int(fd), b, flags); err != nil {
			return false
		}
		return true
	}
	err = s.rc.Read(fun)
	return
}

func (s *Socket) Sendto(b []byte, flags int, to syscall.Sockaddr) (err error) {
	if atomic.LoadUint32(&s.closed) != 0 {
		return os.NewSyscallError("Sendto closed", nil)
	}
	fun := func(fd uintptr) bool {
		if err = syscall.Sendto(int(fd), b, flags, to); err != nil {
			return false
		}
		return true
	}
	err = s.rc.Write(fun)
	return
}

func (s *Socket) Recvmsg(p, oob []byte, flags int) (n, oobn, recvflags int, from syscall.Sockaddr, err error) {
	if atomic.LoadUint32(&s.closed) != 0 {
		return 0, 0, flags, nil, os.NewSyscallError("Recvmsg closed", nil)
	}
	fun := func(fd uintptr) bool {
		if n, oobn, recvflags, from, err = syscall.Recvmsg(int(fd), p, oob, flags); err != nil {
			return false
		}
		return true
	}
	err = s.rc.Read(fun)
	return
}

func (s *Socket) Sendmsg(p, oob []byte, to syscall.Sockaddr, flags int) (n int, err error) {
	if atomic.LoadUint32(&s.closed) != 0 {
		return 0, os.NewSyscallError("Sendmsg closed", nil)
	}
	fun := func(fd uintptr) bool {
		if n, err = syscall.SendmsgN(int(fd), p, oob, to, flags); err != nil {
			return false
		}
		return true
	}
	err = s.rc.Write(fun)
	return
}

func (s *Socket) Bind(sal syscall.Sockaddr) (syscall.Sockaddr, error) {
	if atomic.LoadUint32(&s.closed) != 0 {
		return nil, os.NewSyscallError("BindSockaddrLinklayer closed", nil)
	}
	var err error
	fun := func(fd uintptr) {
		err = syscall.Bind(int(fd), sal)
	}
	if err = s.rc.Control(fun); err == nil {
		fun = func(fd uintptr) {
			if lsa, err := syscall.Getsockname(int(fd)); err == nil {
				sal = lsa
			}
		}
		err = s.rc.Control(fun)
	}
	return sal, err
}

func (s *Socket) SetReadDeadline(d time.Duration) error {
	return s.f.SetReadDeadline(time.Now().Add(d))
}
