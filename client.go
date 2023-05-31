// @@
// @ Author       : Eacher
// @ Date         : 2023-05-31 08:00:36
// @ LastEditTime : 2023-05-31 16:32:35
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /gonmcli/client.go
// @@
package gonmcli

import (
	"fmt"
	"sync"
	"strconv"
	"time"
)

type Client interface {
	WIFIScan(update bool) []WifiInfo
}

type client struct {
	connList 	[]*Connection
	devList 	[]*Device
}

type Connection struct {
	id 			string
	uuid 		string
	_type 		string
	dbus_path 	string
	firmware 	string
	autoconnect bool
	priority 	int32
	ipv4_method string
	ipv4_dns 		string
	ipv4_addresses 	string
	ipv4_gateway 	string
	dev 		*Device
}

type Device struct {
	iface 		string
	_type 		string
	udi 		string
	driver 		string
	firmware 	string
	hw_address 	string
	state 		string
	autoconnect bool
	real 		bool
	software 	bool
	conn 		*Connection
}