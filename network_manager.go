// @@
// @ Author       : Eacher
// @ Date         : 2023-05-24 11:47:01
// @ LastEditTime : 2023-05-29 11:02:04
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /gonmcli/network_manager.go
// @@
package gonmcli
/*
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
	go C.runLoop()
	return nil
}

func NMQuit() error {
	C.quitLoop()
	return nil
}

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
		scanList[i].idx 	=	i
		scanList[i].Ssid 	=	C.GoString(wd.ssid)
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
