# networkmanager

## 简介
	这是一个基于 netlink 开发的 GoLang 以太网管理包。
	
# 例子
```go

func main() {
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
	if err := json.Unmarshal(b, config); err == nil {
		if object, err = networkmanager.InterfaceByName(config.Iface); err != nil {
			fmt.Println("InterfaceByName error", err)
			return
		}
		fmt.Println("Up Interface", object.Up())
		fmt.Println("AddIP", object.AddIP(networkmanager.Addrs{Local: net.ParseIP(config.Address)}))
		if routeList, err := object.RouteList(); err != nil {
			fmt.Println("Get RouteList")
			for _, val := range routeList {
				fmt.Println("Print Route", *val)
			}
			fmt.Println("RouteList End")
		}
		rtmsg := &syscall.RtMsg{
			Table: syscall.RT_TABLE_MAIN, Tos: 0, Protocol: syscall.RTPROT_KERNEL, Type: syscall.RTN_BROADCAST, Scope: syscall.RT_SCOPE_HOST,
		}
		fmt.Println("Gateway AddRoute", object.AddRoute(networkmanager.Routes{RtMsg: rtmsg, Gw: net.ParseIP(config.Gateway)}))
		go func() {
			time.Sleep(time.Second*20)
			fmt.Println("os.Exit(1)")
			os.Exit(1)
		}()
		object.Close()
	}
	fmt.Println("end")
}

```