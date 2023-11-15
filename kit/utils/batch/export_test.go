package batch

func (fp *MultiBatcher[IN, OUT]) Pool() map[string]*Batcher[IN, OUT] {
	return fp.pool
}

func (f *Batcher[IN, OUT]) EntryChan() chan Entry[IN, OUT] {
	return f.entryChan
}
