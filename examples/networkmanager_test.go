// @@
// @ Author       : Eacher
// @ Date         : 2023-02-20 08:50:39
// @ LastEditTime : 2023-07-06 11:24:01
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : Linux networkmanager 使用例子
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /networkmanager/examples/networkmanager_test.go
// @@
package networkmanager_test

import (
	"os"
	"net"
	"log"
	"testing"
	"encoding/json"
	"time"
	"github.com/20yyq/packet"
	"github.com/20yyq/networkmanager"
	"github.com/20yyq/networkmanager/socket/dhcpv4"
)

var josnStr, config = `
	{
		"iface": "eth0",
		"address": "192.168.1.58",
		"gateway": "192.168.1.1"
	}
`, &struct {
	Iface 	string `json:"iface"`
	Address string `json:"address"`
	Gateway string `json:"gateway"`
}{}

func TestLinux(t *testing.T) {
	var object *networkmanager.Interface
	if err := json.Unmarshal([]byte(josnStr), config); err == nil {
		if object, err = networkmanager.InterfaceByName(config.Iface); err != nil {
			t.Log("InterfaceByName error", err)
			return
		}
	}
	t.Log("Up Interface", object.Up())
	// 静态IP
	// custom(object)
	// DHCP获取IP
	dhcp(object)
	object.Close()
	t.Log("end")
}


func custom(manager *networkmanager.Interface) {
	log.Println("AddIP", manager.AddIP(networkmanager.Addrs{Local: net.ParseIP(config.Address)}))
	if routeList, err := manager.RouteList(); err == nil {
		log.Println("Get RouteList")
		for _, val := range routeList {
			log.Println("Print Route", *val)
		}
		log.Println("RouteList End")
	}
	log.Println("Gateway AddRoute", manager.AddRoute(networkmanager.Routes{Gw: net.ParseIP(config.Gateway)}))
	go func() {
		time.Sleep(time.Second*10)
		log.Println("os.Exit(1)")
		os.Exit(1)
	}()
}

func dhcp(manager *networkmanager.Interface) {
	rt := networkmanager.Routes{Gw: net.IP{0,0,0,0}}
	log.Println("AddRoute: ", manager.AddRoute(rt))
	conn, _ := dhcpv4.NewDhcpV4Conn("dhcpv4", packet.IPv4{255,255,255,255})
	dhc1, err := conn.Discover()
	if err != nil {
		log.Println("Discover err: ", err)
		return
	}
	dhc1, err = conn.Request(*dhc1)
	if err != nil {
		log.Println("Request err: ", err)
		return
	}
	for _, v := range dhc1.Options {
		if v.Types == uint8(packet.DHCP_Requested_IP_Address) {
			config.Address = (*packet.IPv4)(v.Value).String()
			log.Println("Address: ", config.Gateway)
		}
		if v.Types == uint8(packet.DHCP_Router) {
			config.Gateway = (*packet.IPv4)(v.Value).String()
			log.Println("Gateway: ", config.Gateway)
		}
	}
	log.Println("RemoveRoute: ", manager.RemoveRoute(rt))
	custom(manager)
}
