package relay

import (
	"sync"
)

type TF struct {
	Counter uint64
	RW      *sync.RWMutex
}

func NewTF() *TF {
	return &TF{Counter: 0, RW: new(sync.RWMutex)}
}
func (tf *TF) Add(val uint64) {
	tf.RW.Lock()
	tf.Counter += val
	tf.RW.Unlock()
}
func (tf *TF) Total() uint64 {
	tf.RW.RLock()
	defer tf.RW.RUnlock()
	return tf.Counter
}
func (tf *TF) Reset() {
	tf.RW.Lock()
	tf.Counter = 0
	tf.RW.Unlock()
}
