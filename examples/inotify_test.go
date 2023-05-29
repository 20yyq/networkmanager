// @@
// @ Author       : Eacher
// @ Date         : 2023-02-20 08:50:39
// @ LastEditTime : 2023-05-29 10:55:36
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : Linux gonmcli 使用例子
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /gonmcli/examples/inotify_test.go
// @@
package gonmcli_test

import (
	"testing"
	"time"
	"github.com/20yyq/gonmcli"
)

func TestLinux(t *testing.T) {
	gonmcli.NMStart()
	for i := 0; i < 100; i++ {
		time.Sleep(time.Second * 5)
		for _, v := range gonmcli.NMScan(false) {
			tmpv := v
			t.Log(tmpv)
		}
	}
	gonmcli.NMQuit()
	time.Sleep(time.Second * 3)
	t.Log("end")
}