// @@
// @ Author       : Eacher
// @ Date         : 2023-05-24 11:47:01
// @ LastEditTime : 2023-06-05 08:33:12
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

	mapClient 	map[*Client]bool
	wifiIdx  	uint8
	wifiChan 	map[uint8]chan []WifiInfo

	err 		error
}

func (cls *baseClient) permissions() map[string]string {
	m := make(map[string]string)
	m[`org.freedesktop.NetworkManager.enable-disable-network`]			= C.GoString(C.Client.permission.ednetwork)
	m[`org.freedesktop.NetworkManager.enable-disable-wifi`]				= C.GoString(C.Client.permission.ednwifi)
	m[`org.freedesktop.NetworkManager.enable-disable-wwan`]				= C.GoString(C.Client.permission.edwwan)
	m[`org.freedesktop.NetworkManager.enable-disable-wimax`]			= C.GoString(C.Client.permission.edwimax)
	m[`org.freedesktop.NetworkManager.sleep-wake`]						= C.GoString(C.Client.permission.sleep_wake)
	m[`org.freedesktop.NetworkManager.network-control`]					= C.GoString(C.Client.permission.network_control)
	m[`org.freedesktop.NetworkManager.wifi.share.protected`]			= C.GoString(C.Client.permission.wifi_protected)
	m[`org.freedesktop.NetworkManager.wifi.share.open`]					= C.GoString(C.Client.permission.wifi_open)
	m[`org.freedesktop.NetworkManager.settings.modify.system`]			= C.GoString(C.Client.permission.modify_system)
	m[`org.freedesktop.NetworkManager.settings.modify.own`]				= C.GoString(C.Client.permission.modify_own)
	m[`org.freedesktop.NetworkManager.settings.modify.hostname`]		= C.GoString(C.Client.permission.modify_hostname)
	m[`org.freedesktop.NetworkManager.settings.modify.global-dns`]		= C.GoString(C.Client.permission.modify_dns)
	m[`org.freedesktop.NetworkManager.reload`]							= C.GoString(C.Client.permission.reload)
	m[`org.freedesktop.NetworkManager.checkpoint-rollback`]				= C.GoString(C.Client.permission.checkpoint)
	m[`org.freedesktop.NetworkManager.enable-disable-statistics`]			= C.GoString(C.Client.permission.edstatic)
	m[`org.freedesktop.NetworkManager.enable-disable-connectivity-check`]	= C.GoString(C.Client.permission.connectivity_check)
	return m
}

func init() {
	if err := clientStart(); err != nil {
		fmt.Println(err)
	}
}

func clientStart() (err error) {
    var gerr *C.GError
    if C.Client.client = C.nm_client_new(nil, &gerr); nil == C.Client.client {
    	err = fmt.Errorf("C client Start error: %s", C.GoString(gerr.message))
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

var scanMutex sync.Mutex
var scanNotify chan bool = nil

func (cls *baseClient) wifiScan() (<-chan []WifiInfo, error) {
	if "yes" != C.GoString(C.Client.permission.ednwifi) && "yes" != C.GoString(C.Client.permission.wifi_protected) && "yes" != C.GoString(C.Client.permission.wifi_open) {
		return nil, fmt.Errorf("permission error")
	}
	scanMutex.Lock()
	defer scanMutex.Unlock()
	if scanNotify != nil {
		return nil, fmt.Errorf("scan busy")
	}
	var wifiChan chan []WifiInfo
	var scanNum int8
	scanNotify, wifiChan, cls.wifiIdx = make(chan bool), make(chan []WifiInfo), cls.wifiIdx + 1
	if _, ok := cls.wifiChan[cls.wifiIdx]; ok {
		delete(cls.wifiChan, cls.wifiIdx)
	}
	cls.wifiChan[cls.wifiIdx] = wifiChan
	for {
		if 1 != C.wifiScanAsync(C.int(cls.wifiIdx)) {
			go func() { scanNotify <- true }()
		}
		if is := <-scanNotify; is {
			fmt.Println("loop")
			time.Sleep(time.Millisecond * 100)
			if scanNum++; 3 < scanNum {
				close(wifiChan)
				close(scanNotify)
				delete(cls.wifiChan, cls.wifiIdx)
				fmt.Println("error")
				break
			}
			continue
		}
		fmt.Println("success")
		break
	}
	scanNotify = nil
	return wifiChan, nil
}

//export scanCallBackFunc
func scanCallBackFunc(idx C.int) {
	if 2 > C.Client.wifiDataLen {
		go func() { scanNotify <- true }()
		return
	}
	close(scanNotify)
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

var eventMutex sync.Mutex
var eventMaps = map[string]bool{}

func (cls *baseClient) getDevEventInfo(devName string) (g_type, g_bssid, g_connId string) {
	eventMutex.Lock()
	if _, ok := eventMaps[devName]; !ok {
		var _type, bssid, connId *C.char
		if 1 == C.notifyDeviceMonitor(C.CString(devName), &_type, &bssid, &connId) {
			g_type, g_bssid, g_connId = C.GoString(_type), C.GoString(bssid), C.GoString(connId)
			C.g_free(C.gpointer(_type))
			C.g_free(C.gpointer(bssid))
			C.g_free(C.gpointer(connId))
			eventMaps[devName] = true
		}
	}
	eventMutex.Unlock()
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
