package batch

import (
	"sync/atomic"
	"time"
	_ "unsafe"

	"github.com/clubpay/ronykit/kit/utils"
)

/*
   Creation Time: 2022 - Jul - 22
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
*/

type NA = struct{}

type Func[IN, OUT any] func(tagID string, entries []Entry[IN, OUT])

type MultiBatcher[IN, OUT any] struct {
	cfg         config
	batcherFunc Func[IN, OUT]
	poolMtx     utils.SpinLock
	pool        map[string]*Batcher[IN, OUT]
}

// NewMulti creates a pool of Batcher functions.
// By calling Enter or EnterAndWait you add the item into the Batcher which
// is identified by 'tagID'.
func NewMulti[IN, OUT any](f Func[IN, OUT], opt ...Option) *MultiBatcher[IN, OUT] {
	cfg := defaultConfig
	for _, o := range opt {
		o(&cfg)
	}

	fp := &MultiBatcher[IN, OUT]{
		cfg:         cfg,
		batcherFunc: f,
		pool:        make(map[string]*Batcher[IN, OUT], 16),
	}

	return fp
}

func (fp *MultiBatcher[IN, OUT]) getBatcher(tagID string) *Batcher[IN, OUT] {
	fp.poolMtx.Lock()
	f := fp.pool[tagID]
	if f == nil {
		f = newBatcher[IN, OUT](fp.batcherFunc, tagID, fp.cfg)
		fp.pool[tagID] = f
	}
	fp.poolMtx.Unlock()

	return f
}

func (fp *MultiBatcher[IN, OUT]) Enter(targetID string, entry Entry[IN, OUT]) {
	fp.getBatcher(targetID).Enter(entry)
}

func (fp *MultiBatcher[IN, OUT]) EnterAndWait(targetID string, entry Entry[IN, OUT]) {
	fp.getBatcher(targetID).EnterAndWait(entry)
}

type Batcher[IN, OUT any] struct {
	spin utils.SpinLock

	readyWorkers int32
	batchSize    int32
	minWaitTime  time.Duration
	flusherFunc  Func[IN, OUT]
	entryChan    chan Entry[IN, OUT]
	tagID        string
}

// NewBatcher construct a new Batcher with tagID. `tagID` is the value that will be passed to
// Func on every batch. This lets you define the same batch func with multiple Batcher objects; MultiBatcher
// is using `tagID` internally to handle different batches of entries in parallel.
func NewBatcher[IN, OUT any](f Func[IN, OUT], tagID string, opt ...Option) *Batcher[IN, OUT] {
	cfg := defaultConfig
	for _, o := range opt {
		o(&cfg)
	}

	return newBatcher[IN, OUT](f, tagID, cfg)
}

func newBatcher[IN, OUT any](f Func[IN, OUT], tagID string, cfg config) *Batcher[IN, OUT] {
	return &Batcher[IN, OUT]{
		readyWorkers: cfg.maxWorkers,
		batchSize:    cfg.batchSize,
		minWaitTime:  cfg.minWaitTime,
		flusherFunc:  f,
		entryChan:    make(chan Entry[IN, OUT], cfg.batchSize),
		tagID:        tagID,
	}
}

func (f *Batcher[IN, OUT]) startWorker() {
	f.spin.Lock()
	if atomic.AddInt32(&f.readyWorkers, -1) < 0 {
		atomic.AddInt32(&f.readyWorkers, 1)
		f.spin.Unlock()

		return
	}
	f.spin.Unlock()

	w := &worker[IN, OUT]{
		f:  f,
		bs: int(f.batchSize),
	}

	go w.run()
}

func (f *Batcher[IN, OUT]) Enter(entry Entry[IN, OUT]) {
	f.entryChan <- entry
	f.startWorker()
}

func (f *Batcher[IN, OUT]) EnterAndWait(entry Entry[IN, OUT]) {
	f.Enter(entry)
	entry.wait()
}

type worker[IN, OUT any] struct {
	f  *Batcher[IN, OUT]
	bs int
}

func (w *worker[IN, OUT]) run() {
	var (
		el        = make([]Entry[IN, OUT], 0, w.bs)
		startTime = utils.NanoTime()
	)
	for {
		for {
			select {
			case e := <-w.f.entryChan:
				el = append(el, e)
				if len(el) < w.bs {
					continue
				}
			default:
			}

			break
		}

		if w.f.minWaitTime > 0 && len(el) < w.bs {
			delta := w.f.minWaitTime - time.Duration(utils.NanoTime()-startTime)
			if delta > 0 {
				time.Sleep(delta)

				continue
			}
		}
		w.f.spin.Lock()
		if len(el) == 0 {
			// clean up and shutdown the worker
			atomic.AddInt32(&w.f.readyWorkers, 1)
			w.f.spin.Unlock()

			break
		}
		w.f.spin.Unlock()
		w.f.flusherFunc(w.f.tagID, el)
		for idx := range el {
			el[idx].done()
		}
		el = el[:0]
	}
}
