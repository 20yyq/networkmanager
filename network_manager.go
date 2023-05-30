// @@
// @ Author       : Eacher
// @ Date         : 2023-05-24 11:47:01
// @ LastEditTime : 2023-05-30 14:46:27
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

func NMStart() error {
	if 1 != C.int(C.init()) {
		return nil
	}
	mapsDevEvent = make(map[string]*DeviceMonitorEvent)
	go C.runLoop()
	return nil
}

func NMQuit() error {
	C.quitLoop()
	return nil
}

/******************************************** WIFI Start ********************************************/

const cycle = 60 * 1
var scanTimestamp int64
var scanList []WifiInfo
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

var scanMutex sync.Mutex
var scanNotify chan bool = nil
func NMScan(update bool) []WifiInfo {
	scanMutex.Lock()
	defer scanMutex.Unlock()
	if update || (scanTimestamp + cycle) < time.Now().Unix() {
		scanTimestamp = time.Now().Unix()
		scanNum := 0
		scanNotify = make(chan bool)
		for {
			if 1 != C.wifiScanAsync() {
				go func() { scanNotify <- true }()
			}
			if is := <-scanNotify; is {
				fmt.Println("loop")
				time.Sleep(time.Millisecond * 100)
				if scanNum++; 3 < scanNum {
					close(scanNotify)
					fmt.Println("error")
					break
				}
				continue
			}
			fmt.Println("success")
			break
		}
		scanNotify = nil
	}
	l := make([]WifiInfo, len(scanList))
	copy(l, scanList)
	return l
}

//export scanCallBackFunc
func scanCallBackFunc(name *C.char, n C.guint, wd *C.WifiData) C.int {
	i := uint(C.uint(n))
	switch C.GoString(name) {
	case "start":
		if 2 > i {
			i = 0
			go func() { scanNotify <- true }()
			break
		}
		scanList = make([]WifiInfo, i)
		i = 1
	case "runFunc":
		scanList[i].idx, scanList[i].Ssid =	i, "nil"
		if wd.ssid != nil {
			scanList[i].Ssid=	C.GoString(wd.ssid)
		}
		scanList[i].Bssid 	=	C.GoString(wd.bssid)
		scanList[i].Mode 	=	C.GoString(wd.mode)
		scanList[i].Flags 	=	uint8(C.int(wd.flags))
		scanList[i].Strength=	uint8(C.int(wd.strength))
		scanList[i].Freq 	=	strconv.FormatInt(int64(C.uint(wd.freq)), 10) + " MHz"
		scanList[i].Bitrate =	strconv.FormatInt(int64(C.uint(wd.bitrate) / 1000), 10) + " Mbit/s"
		scanList[i].dBusPath=	C.GoString(wd.dbus_path)
	case "close":
		close(scanNotify)
	default:

	}
	return C.int(i)
}

/******************************************** WIFI End ********************************************/

/******************************************** Device Start ********************************************/


type devEvent struct {
	TimeFormat 	string
	FuncName 	string
	Flags 		uint32
}

type DeviceMonitorEvent struct {
	dev 	string
	_type 	string
	bssid 	string
	connId 	string
	echan 	chan devEvent
}

var devEventMutex sync.RWMutex
var mapsDevEvent map[string]*DeviceMonitorEvent

//export deviceMonitorCallBackFunc
func deviceMonitorCallBackFunc(funcName *C.char, devName *C.char, n C.guint) {
	go func(f, d string, i uint32) {
		devEventMutex.RLock()
		val, _ := mapsDevEvent[d]
		devEventMutex.RUnlock()
		if val != nil {
			val.echan <- devEvent{TimeFormat: time.Now().Format(time.DateTime), FuncName: f, Flags: i}
		}
	}(C.GoString(funcName), C.GoString(devName), uint32(C.uint(n)))
}

func NewDevEvent(devName string) *DeviceMonitorEvent {
	devEventMutex.RLock()
	val, _ := mapsDevEvent[devName]
	devEventMutex.RUnlock()
	devEventMutex.Lock()
	defer devEventMutex.Unlock()
	if nil == val {
		var _type, bssid, connId *C.char
		if 1 != C.notifyDeviceMonitor(C.CString(devName), _type, bssid, connId) {
			return nil
		}
		val = &DeviceMonitorEvent{dev: devName, echan: make(chan devEvent, 10)}
		val._type, val.bssid, val.connId = C.GoString(_type), C.GoString(bssid), C.GoString(connId)
		C.g_free(C.gpointer(_type))
		C.g_free(C.gpointer(bssid))
		C.g_free(C.gpointer(connId))
		mapsDevEvent[devName] = val
	}
	return val
}

func (dme *DeviceMonitorEvent) Event() string {
	if de, ok := <- dme.echan; ok {
		switch de.Flags {
		case NM_DEVICE_STATE_UNMANAGED:
			return fmt.Sprintf("%s the device is recognized, but not managed by NetworkManager", de.TimeFormat)
		case NM_DEVICE_STATE_UNAVAILABLE:
			return fmt.Sprintf("%s the device is managed by NetworkManager, but is not available for use", de.TimeFormat)
		case NM_DEVICE_STATE_DISCONNECTED:
			return fmt.Sprintf("%s the device can be activated, but is currently idle and not connected to a network", de.TimeFormat)
		case NM_DEVICE_STATE_PREPARE:
			return fmt.Sprintf("%s the device is preparing the connection to the network", de.TimeFormat)
		case NM_DEVICE_STATE_CONFIG:
			return fmt.Sprintf("%s the device is connecting to the requested network", de.TimeFormat)
		case NM_DEVICE_STATE_NEED_AUTH:
			return fmt.Sprintf("%s the device requires more information to continue connecting to the requested network", de.TimeFormat)
		case NM_DEVICE_STATE_IP_CONFIG:
			return fmt.Sprintf("%s the device is requesting IPv4 and/or IPv6 addresses and routing information from the network", de.TimeFormat)
		case NM_DEVICE_STATE_IP_CHECK:
			return fmt.Sprintf("%s the device is checking whether further action is required for the requested network connection", de.TimeFormat)
		case NM_DEVICE_STATE_SECONDARIES:
			return fmt.Sprintf(`%s the device is waiting for a secondary 
				connection (like a VPN) which must activated before the device can be activated`, de.TimeFormat)
		case NM_DEVICE_STATE_ACTIVATED:
			return fmt.Sprintf("%s the device has a network connection, either local or global", de.TimeFormat)
		case NM_DEVICE_STATE_DEACTIVATING:
			return fmt.Sprintf("%s the network connection may still be valid", de.TimeFormat)
		case NM_DEVICE_STATE_FAILED:
			return fmt.Sprintf("%s the device failed to connect to the requested network and is cleaning up the connection request", de.TimeFormat)
		case NM_DEVICE_STATE_UNKNOWN:
			fallthrough
		default:
			return fmt.Sprintf("%s the device's state is unknown", de.TimeFormat)
		}
	}
	return fmt.Sprintf("%s Event close", time.Now().Format(time.DateTime))
}


/******************************************** Device End ********************************************/