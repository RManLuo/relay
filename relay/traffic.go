package relay

import (
	"sync"
)

type TF struct {
	TCP_UP   uint64
	TCP_DOWN uint64
	UDP      uint64
	RW       *sync.RWMutex
}

func NewTF() *TF {
	return &TF{TCP_UP: 0, TCP_DOWN: 0, UDP: 0, RW: new(sync.RWMutex)}
}

func (tf *TF) Total() uint64 {
	tf.RW.RLock()
	defer tf.RW.RUnlock()
	return tf.TCP_UP + tf.TCP_DOWN + tf.UDP
}
func (tf *TF) Reset() {
	tf.RW.Lock()
	tf.TCP_UP, tf.TCP_DOWN, tf.UDP = 0, 0, 0
	tf.RW.Unlock()
}
