//golangcitest:args -Eforbidigo
//golangcitest:config_path testdata/configs/forbidigo.yml
package testdata

import (
	"fmt"
	"time"
)

func Forbidigo() {
	fmt.Printf("too noisy!!!")  // want "use of `fmt\\.Printf` forbidden by pattern `fmt\\\\.Print\\.\\*`"
	time.Sleep(time.Nanosecond) // want "no sleeping!"
}
