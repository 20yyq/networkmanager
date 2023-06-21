// @@
// @ Author       : Eacher
// @ Date         : 2023-02-20 08:50:39
// @ LastEditTime : 2023-06-21 08:33:27
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : Linux networkmanager 使用例子
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /networkmanager/examples/networkmanager_test.go
// @@
package gonmcli_test

import (
	"testing"
	"time"
	"github.com/20yyq/networkmanager"
)

func TestLinux(t *testing.T) {
	object, err := InterfaceByName("eth0")
	if err != nil {
		t.Log("InterfaceByName error", err)
		return
	}
	t.Log("object.list", object.list, object.sock.Pid)
	// t.Log("Up", object.Up())
	// time.Sleep(time.Second*5)
	// t.Log("RemoveIP", object.RemoveIP(Addrs{Local: net.IPv4(192, 168, 1, 111).To4()}))
	// t.Log("AddIP", object.AddIP(Addrs{Local: net.IPv4(192, 168, 1, 111).To4()}))
	// t.Log("ReplaceIP", object.ReplaceIP(Addrs{Local: net.IPv4(192, 168, 1, 110).To4()}))
	// t.Log("Down", object.Down())
	l, err := object.IPList()
	t.Log("IPList ", l, err, object.iface.Name)
	time.Sleep(time.Second*5)
	object.Close()
	t.Log("object.list", object.list)
	<-object.closes
	// time.Sleep(time.Second*5)
	t.Log("end")
}