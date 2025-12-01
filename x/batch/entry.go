package batch

type Entry[IN, OUT any] interface {
	wait()
	done()
	Value() IN
	Callback(out OUT)
}

type entry[IN, OUT any] struct {
	v  IN
	ch chan struct{}
	cb func(OUT)
}

func NewEntry[IN, OUT any](v IN, callbackFn func(out OUT)) Entry[IN, OUT] {
	return &entry[IN, OUT]{
		v:  v,
		cb: callbackFn,
		ch: make(chan struct{}, 1),
	}
}

func (e *entry[IN, OUT]) wait() {
	<-e.ch
}

func (e *entry[IN, OUT]) done() {
	e.ch <- struct{}{}
}

func (e *entry[IN, OUT]) Value() IN {
	return e.v
}

func (e *entry[IN, OUT]) Callback(out OUT) {
	if e.cb != nil {
		e.cb(out)
	}
}
