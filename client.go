// @@
// @ Author       : Eacher
// @ Date         : 2023-05-31 08:00:36
// @ LastEditTime : 2023-06-03 16:07:40
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
	"time"
)

type Client struct {
	*baseClient
	closes  		bool
	eMutex 			sync.RWMutex
	mapDevEvent 	map[string]*DeviceMonitorEvent
	sMutex 			sync.RWMutex
	scanList 		[]WifiInfo
	scanTimestamp 	int64
	ScanCycle 		time.Duration
}

func NewClient() (*Client, error) {
	if primary == nil {
		if err := clientStart(); err != nil {
			return nil, err
		}
	}
	primary.clientMutex.Lock()
	defer primary.clientMutex.Unlock()
	c := &Client{
		baseClient: primary, mapDevEvent: make(map[string]*DeviceMonitorEvent),
		ScanCycle: time.Duration(time.Minute),
	}
	primary.mapClient[c] = true
	return c, nil
}

func (c *Client) WifiScan(update bool) []WifiInfo {
	if update || (c.scanTimestamp + int64(c.ScanCycle.Seconds())) < time.Now().Unix() {
		c.sMutex.Lock()
		c.scanTimestamp = time.Now().Unix()
		ch, err := c.wifiScan()
		if err != nil {
			fmt.Println("error", err)
			c.sMutex.Unlock()
			return []WifiInfo{}
		}
		c.scanList = <-ch
		c.sMutex.Unlock()
	}
	c.sMutex.RLock()
	l := make([]WifiInfo, len(c.scanList))
	copy(l, c.scanList)
	c.sMutex.RUnlock()
	return l
}

func (c *Client) NewDevEvent(devName string) *DeviceMonitorEvent {
	_type, bssid, connId := c.getDevEventInfo(devName)
	c.eMutex.Lock()
	defer c.eMutex.Unlock()
	dEvent, _ := c.mapDevEvent[devName]
	if dEvent == nil && bssid != "" {
		dEvent = &DeviceMonitorEvent{dev: devName, _type: _type, bssid: bssid, connId: connId, echan: make(chan devEvent, 10)}
		c.mapDevEvent[devName] = dEvent
	}
	return dEvent
}

func (c *Client) RemoveDevEvent(devName string) {
	c.eMutex.Lock()
	defer c.eMutex.Unlock()
	dEvent, _ := c.mapDevEvent[devName]
	if dEvent != nil {
		for {
			if 10 == len(dEvent.echan) {
				<-dEvent.echan
				continue
			}
			break
		}
		close(dEvent.echan)
		delete(c.mapDevEvent, devName)
	}
}

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

func (dme *DeviceMonitorEvent) Event() string {
	if de, ok := <-dme.echan; ok {
		return fmt.Sprintf("%s %s", de.TimeFormat, de.State)
	}
	return fmt.Sprintf("%s Event close", time.Now().Format(time.DateTime))
}
