package limits

import (
	"runtime"
	"syscall"
)

func Raise() error {
	var l syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &l); err != nil {
		return err
	}
	if runtime.GOOS == "darwin" && l.Cur < 10240 {
		l.Cur = 10240
	}
	if runtime.GOOS != "darwin" && l.Cur < 60000 {
		if l.Max < 60000 {
			l.Max = 60000 // with CAP_SYS_RESOURCE capability
		}
		l.Cur = l.Max
	}
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &l); err != nil {
		return err
	}
	return nil
}
