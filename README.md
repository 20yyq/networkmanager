# networkmanager

## 简介
	这是一个基于 netlink 开发的 GoLang 以太网管理包。
	
# 例子
```go

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

func main() {
	var object *networkmanager.Interface
	if err := json.Unmarshal([]byte(josnStr), config); err == nil {
		if object, err = networkmanager.InterfaceByName(config.Iface); err != nil {
			fmt.Println("InterfaceByName error", err)
			return
		}
	}
	fmt.Println("Up Interface", object.Up())
	// 静态IP
	// custom(object)
	// DHCP获取IP
	dhcp(object)
	manager.Close()
	fmt.Println("end")
}

func custom(manager *networkmanager.Interface) {
	fmt.Println("AddIP", manager.AddIP(networkmanager.Addrs{Local: net.ParseIP(config.Address)}))
	if routeList, err := manager.RouteList(); err == nil {
		fmt.Println("Get RouteList")
		for _, val := range routeList {
			fmt.Println("Print Route", *val)
		}
		fmt.Println("RouteList End")
	}
	fmt.Println("Gateway AddRoute", manager.AddRoute(networkmanager.Routes{Gw: net.ParseIP(config.Gateway)}))
	go func() {
		time.Sleep(time.Second*10)
		fmt.Println("os.Exit(1)")
		os.Exit(1)
	}()
}

func dhcp(manager *networkmanager.Interface) {
	rt := networkmanager.Routes{Gw: net.IP{0,0,0,0}}
	fmt.Println("AddRoute: ", manager.AddRoute(rt))
	conn, _ := dhcpv4.NewDhcpV4Conn("dhcpv4", packet.IPv4{255,255,255,255})
	dhc1, err := conn.Discover()
	if err != nil {
		fmt.Println("Discover err: ", err)
		return
	}
	dhc1, err = conn.Request(*dhc1)
	if err != nil {
		fmt.Println("Request err: ", err)
		return
	}
	for _, v := range dhc1.Options {
		if v.Types == uint8(packet.DHCP_Requested_IP_Address) {
			config.Address = (*packet.IPv4)(v.Value).String()
			fmt.Println("Address: ", config.Gateway)
		}
		if v.Types == uint8(packet.DHCP_Router) {
			config.Gateway = (*packet.IPv4)(v.Value).String()
			fmt.Println("Gateway: ", config.Gateway)
		}
	}
	fmt.Println("RemoveRoute: ", manager.RemoveRoute(rt))
	custom(manager)
}

```