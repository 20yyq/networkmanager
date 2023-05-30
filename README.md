# inotify
gonmcli
## 简介
	这是一个基于 NetworkManager 、 libnm 开发的 GoLang 网络管理客户端，实现WIFI管理和网卡设备监听等功能。
	
# 例子
```go

func main() {
	w, err := inotify.NewWatcher()
	if err != nil {
		fmt.Println("NewWatcher err", err)
		return
	}
	w.AddWatch("/temp", syscall.IN_OPEN|syscall.IN_CLOSE|syscall.IN_DELETE|syscall.IN_DELETE_SELF|syscall.IN_CREATE|syscall.IN_IGNORED|syscall.IN_MODIFY|syscall.IN_MOVE|syscall.IN_MOVE_SELF|syscall.IN_MOVED_FROM|syscall.IN_MOVED_TO|syscall.IN_MOVE_SELF|syscall.IN_ATTRIB)
	fmt.Println("start")
	for {
		ws, err := w.WaitEvent()
		if err != nil {
			fmt.Println("WaitEvent Error", err)
			time.Sleep(time.Millisecond*300)
			continue
		}
		fmt.Println("WaitEvent:", ws.Mask, ws.FileName, ws.GetEventName())
	}
	fmt.Println("end")
}

```