# networkmanager

## 简介
	这是一个基于 netlink 开发的 GoLang 以太网管理包。
	
# 例子
```go

func main() {
	object, _ := InterfaceByName("eth0")
	fmt.Println("object.list", object.list, object.sock.Pid)
	// fmt.Println("num1", object.Up())
	// time.Sleep(time.Second*5)
	// fmt.Println("RemoveIP", object.RemoveIP(Addrs{Local: net.IPv4(192, 168, 1, 111).To4()}))
	// fmt.Println("AddIP", object.AddIP(Addrs{Local: net.IPv4(192, 168, 1, 111).To4()}))
	// fmt.Println("ReplaceIP", object.ReplaceIP(Addrs{Local: net.IPv4(192, 168, 1, 110).To4()}))
	// fmt.Println("num2", object.Down())
	l, err := object.IPList()
	fmt.Println("IPList ", l, err, object.iface.Name)
	time.Sleep(time.Second*5)
	object.Close()
	fmt.Println("object.list", object.list)
	<-object.closes
	// time.Sleep(time.Second*5)
	fmt.Println("end")
}

```