package setup

import "time"

func waitCheck(timeout time.Duration, check func() error) error {
	deadline := time.Now().Add(timeout)
	var err error
	for err = check(); err != nil && !time.Now().After(deadline); err = check() {
		time.Sleep(time.Second)
	}
	return err
}
