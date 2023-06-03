// @@
// @ Author       : Eacher
// @ Date         : 2023-05-24 11:47:01
// @ LastEditTime : 2023-06-03 16:11:30
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

var primary *baseClient

type baseClient struct {
	conn 		[]*C.ConnData
	device 		[]*C.DevData

	clientMutex	sync.RWMutex
	eventMutex 	sync.RWMutex
	scanMutex 	sync.Mutex

	mapClient 	map[*Client]bool
	mapEvent 	map[string]bool
	wifiChan 	map[uint8]chan WifiInfo

	scanList 	[]WifiInfo
	scanNotify 	chan bool
	idx  		uint8
	err 		error
}

func init() {
	if err := clientStart(); err != nil {
		fmt.Println("clientStart error: ", err)
	}
}

func clientStart() (err error) {
    var gerr *C.GError
    if C.Client.client = C.nm_client_new(nil, &gerr); nil == C.Client.client {
    	err = fmt.Errorf("%s", C.GoString(gerr.message))
        C.g_error_free(gerr)
        return
    }
    if C.int(C.nm_client_get_nm_running(C.Client.client)) != 1 {
        C.g_object_unref(C.gpointer(C.Client.client))
        C.Client.client = nil
    	return fmt.Errorf("clientStart error")
    }
	primary = &baseClient{
		mapClient: make(map[*Client]bool),
		mapEvent: make(map[string]bool),
		wifiChan: make(map[uint8]chan []WifiInfo),
	}
    C.Client.loop = C.g_main_loop_new(nil, C.gboolean(0))
	go C.init()
	return
}

func ClientQuit() error {
	C.g_main_loop_quit(C.Client.loop)
	return nil
}

//export initCallBackFunc
func initCallBackFunc() {
	primary.conn = make([]*C.ConnData, C.Client.connDataLen)
	for i := 0; i < int(C.Client.connDataLen); i++ {
		primary.conn[i] = C.getConnData(C.int(i))
	}
	primary.device = make([]*C.DevData, C.Client.devDataLen)
	for i := 0; i < int(C.Client.devDataLen); i++ {
		primary.device[i] = C.getDevData(C.int(i))
	}
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

func (cls *baseClient) wifiScan() (<-chan []WifiInfo, error) {
	if 1 != C.Client.permission.ednwifi && 1 != C.Client.permission.wifi_protected && 1 != C.Client.permission.wifi_open {
		return nil, fmt.Errorf("permission error")
	}
	cls.scanMutex.Lock()
	defer cls.scanMutex.Unlock()
	if cls.scanNotify != nil {
		return nil, fmt.Errorf("scan busy")
	}
	var wifiChan chan []WifiInfo
	var scanNum int8
	cls.scanNotify, wifiChan, cls.idx = make(chan bool), make(chan []WifiInfo), cls.idx + 1
	if _, ok := cls.wifiChan[cls.idx]; ok {
		delete(cls.wifiChan, cls.idx)
	}
	cls.wifiChan[cls.idx] = wifiChan
	for {
		if 1 != C.wifiScanAsync(C.int(cls.idx)) {
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
	return wifiChan, nil
}

//export scanCallBackFunc
func scanCallBackFunc(idx C.int) {
	if 1 < C.Client.wifiDataLen {
		close(primary.scanNotify)
		if wifiChan, ok := primary.wifiChan[uint8(idx)]; ok && wifiChan != nil {
			list := make([]WifiInfo, 0, C.Client.wifiDataLen)
			for i := 0; i < int(C.Client.wifiDataLen); i++ {
				if wd := C.getWifiData(C.int(i)); wd != nil {
					var info WifiInfo
					info.idx, info.Ssid =	uint(i), "nil"
	                if wd.ssid != nil {
	                    info.Ssid = C.GoString(wd.ssid)
	                    C.g_free(C.gpointer(wd.ssid))
	                }
					info.Bssid 		=	C.GoString(wd.bssid)
					info.Mode 		=	C.GoString(wd.mode)
					info.Flags 		=	uint8(C.int(wd.flags))
					info.Strength 	=	uint8(C.int(wd.strength))
					info.Freq 		=	strconv.FormatInt(int64(C.uint(wd.freq)), 10) + " MHz"
					info.Bitrate 	=	strconv.FormatInt(int64(C.uint(wd.bitrate) / 1000), 10) + " Mbit/s"
					info.dBusPath 	=	C.GoString(wd.dbus_path)
					list = append(list, info)
				}
			}
			wifiChan <- list
			close(wifiChan)
		}
		return
	}
	go func() { primary.scanNotify <- true }()
}

/******************************************** WIFI End ********************************************/

/******************************************** Device Start ********************************************/

//export deviceMonitorCallBackFunc
func deviceMonitorCallBackFunc(funcName *C.char, devName *C.char, n C.guint) {
	f, d, i := C.GoString(funcName), C.GoString(devName), uint32(C.uint(n))
	primary.clientMutex.RLock()
	defer primary.clientMutex.RUnlock()
	for cli, _ := range primary.mapClient {
		cli.eMutex.RLock()
		if event, _ := cli.mapDevEvent[d]; event != nil {
			go func(event *DeviceMonitorEvent, f string, i uint32) {
				event.echan <- devEvent{TimeFormat: time.Now().Format(time.DateTime), FuncName: f, Flags: i}
			}(event, f, i)
		}
		cli.eMutex.RUnlock()
	}
}

func (cls *baseClient) getDevEventInfo(devName string) (string, string, string) {
	cls.eventMutex.Lock()
	_, ok := cls.mapEvent[devName]
	var g_type, g_bssid, g_connId string
	if !ok {
		var _type, bssid, connId *C.char
		if 1 == C.notifyDeviceMonitor(C.CString(devName), &_type, &bssid, &connId) {
			g_type, g_bssid, g_connId = C.GoString(_type), C.GoString(bssid), C.GoString(connId)
			C.g_free(C.gpointer(_type))
			C.g_free(C.gpointer(bssid))
			C.g_free(C.gpointer(connId))
			cls.mapEvent[devName] = true
		}
	}
	cls.eventMutex.Unlock()
	if g_bssid == "" {
		cls.clientMutex.RLock()
		defer cls.clientMutex.RUnlock()
		for val, _ := range cls.mapClient {
			if de, _ := val.mapDevEvent[devName]; de != nil {
				g_type, g_bssid, g_connId = de._type, de.bssid, de.connId
				break
			}
		}
	}
	return g_type, g_bssid, g_connId
}

/******************************************** Device End ********************************************/