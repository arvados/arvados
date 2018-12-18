package arvados

type throttle struct {
	c chan struct{}
}

func newThrottle(n int) *throttle {
	return &throttle{c: make(chan struct{}, n)}
}

func (t *throttle) Acquire() {
	t.c <- struct{}{}
}

func (t *throttle) Release() {
	<-t.c
}
