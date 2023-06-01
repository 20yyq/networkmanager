// @@
// @ Author       : Eacher
// @ Date         : 2023-05-24 11:47:01
// @ LastEditTime : 2023-06-01 15:47:09
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /gonmcli/network_manager.go
// @@
package gonmcli
/*
#cgo linux pkg-config: libnm
#cgo CFLAGS: -I.
#cgo LDFLAGS: -L${SRCDIR} -lgonmcli
#include "network.h"
*/
import "C"
import (
	"fmt"
	"sync"
	"strconv"
	"time"
)

const cycle = 60 * 1
var scanTimestamp int64
var primaryClient *clients

type clients struct {
	primary 	*client
	rwmutex		sync.RWMutex
	scanMutex 	sync.Mutex
	scanList 	[]WifiInfo
	scanNotify 	chan bool
	clientMap 	map[*client]bool
	eventMutex 	sync.RWMutex
	eventMap 	map[string]*DeviceMonitorEvent
}

func NMStart() error {
	if 1 != C.int(C.init()) {
		return nil
	}
	primaryClient = &clients{
		primary: &client{
			connList: make([]*Connection, 0),
			devList: make([]*Device, 0),
		},
		clientMap: make(map[*client]bool),
		eventMap: make(map[string]*DeviceMonitorEvent),
	}
	go C.runLoop()
	return nil
}

func NMQuit() error {
	C.quitLoop()
	return nil
}

//export setConnectionFunc
func setConnectionFunc(data *C.ConnData) {
	conn := &Connection{
		id:		C.GoString(data.id),
		uuid:	C.GoString(data.uuid),
		_type:	C.GoString(data._type),
		dbus_path:		C.GoString(data.dbus_path),
		firmware:		C.GoString(data.firmware),
		priority:		int32(data.priority),
		ipv4_method:	C.GoString(data.ipv4_method),
		ipv4_dns:		C.GoString(data.ipv4_dns),
		ipv4_addresses:	C.GoString(data.ipv4_addresses),
		ipv4_gateway:	C.GoString(data.ipv4_gateway),
	}
	if 1 == C.int(data.autoconnect) {
		conn.autoconnect = true
	}
	primaryClient.primary.connList = append(primaryClient.primary.connList, conn)
}

//export setDeviceFunc
func setDeviceFunc(data *C.DevData) {
	dev := &Device{
		iface:		C.GoString(data.iface),
		_type:		C.GoString(data._type),
		udi:		C.GoString(data.udi),
		driver:		C.GoString(data.driver),
		firmware:	C.GoString(data.firmware),
		hw_address:	C.GoString(data.hw_address),
		state:		C.GoString(data.state),
	}
	if 1 == C.int(data.autoconnect) {
		dev.autoconnect = true
	}
	if 1 == C.int(data.real) {
		dev.real = true
	}
	if 1 == C.int(data.software) {
		dev.software = true
	}
	if uuid := C.GoString(data.uuid); uuid != "" {
		for i := 0; i < len(primaryClient.primary.connList); i++ {
			if primaryClient.primary.connList[i].uuid == uuid {
				dev.conn = primaryClient.primary.connList[i]
				primaryClient.primary.connList[i].dev = dev
				break
			}
		}
	}
	primaryClient.primary.devList = append(primaryClient.primary.devList, dev)
}

/******************************************** WIFI Start ********************************************/

type WifiInfo struct {
	idx 		uint
	dBusPath 	string
	Ssid 		string	`json:"ssid"`
	Bssid 		string	`json:"bssid"`
	Mode 		string	`json:"mode"`
	Flags 		uint8	`json:"flags"`
	Strength 	uint8	`json:"strength"`
	Freq 		string	`json:"freq"`
	Bitrate 	string	`json:"bitrate"`
}

func (cls *clients) wifiScan(update bool) []WifiInfo {
	cls.scanMutex.Lock()
	defer cls.scanMutex.Unlock()
	if update || (scanTimestamp + cycle) < time.Now().Unix() {
		scanTimestamp = time.Now().Unix()
		if 1 == C.Permission.ednwifi || 1 == C.Permission.wifi_protected || 1 == C.Permission.wifi_open {
			scanNum := 0
			cls.scanNotify = make(chan bool)
			for {
				if 1 != C.wifiScanAsync() {
					go func() { cls.scanNotify <- true }()
				}
				if is := <-cls.scanNotify; is {
					fmt.Println("loop")
					time.Sleep(time.Millisecond * 100)
					if scanNum++; 3 < scanNum {
						close(cls.scanNotify)
						fmt.Println("error")
						break
					}
					continue
				}
				fmt.Println("success")
				break
			}
			cls.scanNotify = nil
		}
	}
	l := make([]WifiInfo, len(cls.scanList))
	copy(l, cls.scanList)
	return l
}

//export scanCallBackFunc
func scanCallBackFunc(name *C.char, n C.guint, wd *C.WifiData) C.int {
	i := uint(C.uint(n))
	switch C.GoString(name) {
	case "start":
		if 2 > i {
			i = 0
			go func() { primaryClient.scanNotify <- true }()
			break
		}
		primaryClient.scanList = make([]WifiInfo, i)
		i = 1
	case "runFunc":
		primaryClient.scanList[i].idx, primaryClient.scanList[i].Ssid =	i, "nil"
		if wd.ssid != nil {
			primaryClient.scanList[i].Ssid=	C.GoString(wd.ssid)
		}
		primaryClient.scanList[i].Bssid 	=	C.GoString(wd.bssid)
		primaryClient.scanList[i].Mode 	=	C.GoString(wd.mode)
		primaryClient.scanList[i].Flags 	=	uint8(C.int(wd.flags))
		primaryClient.scanList[i].Strength=	uint8(C.int(wd.strength))
		primaryClient.scanList[i].Freq 	=	strconv.FormatInt(int64(C.uint(wd.freq)), 10) + " MHz"
		primaryClient.scanList[i].Bitrate =	strconv.FormatInt(int64(C.uint(wd.bitrate) / 1000), 10) + " Mbit/s"
		primaryClient.scanList[i].dBusPath=	C.GoString(wd.dbus_path)
	case "close":
		close(primaryClient.scanNotify)
	default:

	}
	return C.int(i)
}

/******************************************** WIFI End ********************************************/

/******************************************** Device Start ********************************************/

type devEvent struct {
	TimeFormat 	string
	FuncName 	string
	State 		string
	Flags 		uint32
}

type DeviceMonitorEvent struct {
	dev 	string
	_type 	string
	bssid 	string
	connId 	string
	echan 	chan devEvent
}

//export deviceMonitorCallBackFunc
func deviceMonitorCallBackFunc(funcName *C.char, devName *C.char, state *C.char, n C.guint) {
	go func(f, d, s string, i uint32) {
		primaryClient.eventMutex.RLock()
		val, _ := primaryClient.eventMap[d]
		primaryClient.eventMutex.RUnlock()
		if val != nil {
			val.echan <- devEvent{TimeFormat: time.Now().Format(time.DateTime), FuncName: f, State: s, Flags: i}
		}
	}(C.GoString(funcName), C.GoString(devName), C.GoString(state), uint32(C.uint(n)))
}

func (cls *clients) removeDevEvent(devName string) {
	cls.eventMutex.Lock()
	val, _ := cls.eventMap[devName]
	defer cls.eventMutex.Unlock()
	if nil != val {
		C.removeDeviceMonitor(C.CString(val.dev))
		for {
			if 10 == len(val.echan) {
				<-val.echan
				continue
			}
			break
		}
		close(val.echan)
		delete(cls.eventMap, val.dev)
	}
}

func (cls *clients) newDevEvent(devName string) *DeviceMonitorEvent {
	cls.eventMutex.Lock()
	val, _ := cls.eventMap[devName]
	defer cls.eventMutex.Unlock()
	if nil == val {
		var _type, bssid, connId *C.char
		if 1 != C.notifyDeviceMonitor(C.CString(devName), &_type, &bssid, &connId) {
			return nil
		}
		val = &DeviceMonitorEvent{dev: devName, echan: make(chan devEvent, 10)}
		val._type, val.bssid, val.connId = C.GoString(_type), C.GoString(bssid), C.GoString(connId)
		C.g_free(C.gpointer(_type))
		C.g_free(C.gpointer(bssid))
		C.g_free(C.gpointer(connId))
		cls.eventMap[devName] = val
	}
	return val
}

func (dme *DeviceMonitorEvent) Event() string {
	if de, ok := <-dme.echan; ok {
		return fmt.Sprintf("%s %s", de.TimeFormat, de.State)
	}
	return fmt.Sprintf("%s Event close", time.Now().Format(time.DateTime))
}


/******************************************** Device End ********************************************/