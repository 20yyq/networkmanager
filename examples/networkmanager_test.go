// @@
// @ Author       : Eacher
// @ Date         : 2023-02-20 08:50:39
// @ LastEditTime : 2023-06-28 15:40:38
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : Linux networkmanager 使用例子
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /networkmanager/examples/networkmanager_test.go
// @@
package networkmanager_test

import (
	"testing"
	"syscall"
	"encoding/json"
	"time"
	"github.com/20yyq/networkmanager"
)

func TestLinux(t *testing.T) {
	josnStr, config := `
		{
			"iface": "eth0",
			"address": "192.168.1.10",
			"gateway": "192.168.1.1"
		}
	`, &struct {
		Iface 	string `json:"iface"`
		Address string `json:"address"`
		Gateway string `json:"gateway"`
	}{}
	var object *networkmanager.Interface
	if err := json.Unmarshal([]byte(josnStr), config); err == nil {
		if object, err = networkmanager.InterfaceByName(config.Iface); err != nil {
			t.Log("InterfaceByName error", err)
			return
		}
		t.Log("Up Interface", object.Up())
		t.Log("AddIP", object.AddIP(networkmanager.Addrs{Local: net.ParseIP(config.Address)}))
		if routeList, err := object.RouteList(); err != nil {
			t.Log("Get RouteList")
			for _, val := range routeList {
				t.Log("Print Route", *val)
			}
			t.Log("RouteList End")
		}
		rtmsg := &syscall.RtMsg{
			Table: syscall.RT_TABLE_MAIN, Tos: 0, Protocol: syscall.RTPROT_KERNEL, Type: syscall.RTN_BROADCAST, Scope: syscall.RT_SCOPE_HOST,
		}
		t.Log("Gateway AddRoute", object.AddRoute(networkmanager.Routes{RtMsg: rtmsg, Gw: net.ParseIP(config.Gateway)}))
		go func() {
			time.Sleep(time.Second*20)
			t.Log("os.Exit(1)")
			os.Exit(1)
		}()
		object.Close()
	}
	t.Log("end")
}