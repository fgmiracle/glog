package glog

import (
	"errors"
)

const (
	defaultSize     = 1024
	messageOverload = 1024
)

var (
	nilMsg interface{}
)

type queue struct {
	lock              SpinLock
	cap               int
	head              int
	tail              int
	overload          int
	overloadThreshold int
	message           []interface{}
}

func newQueue() *queue {
	q := new(queue)
	q.cap = defaultSize
	q.tail = 0
	q.head = 0
	q.message = make([]interface{}, q.cap)
	q.lock.Init()
	q.overload = 0
	q.overloadThreshold = messageOverload
	return q
}

func (mq *queue) Clear() {
	mq.message = make([]interface{}, mq.cap)
}

func (mq *queue) Len() uint32 {
	var head, tail, cap int

	mq.lock.Lock()
	head = mq.head
	tail = mq.tail
	cap = mq.cap
	mq.lock.Unlock()

	if head <= tail {
		return uint32(tail - head)
	}

	return uint32(tail + cap - head)
}

func (mq *queue) expandQueue() {
	newMessage := make([]interface{}, mq.cap*2)
	for i := 0; i < mq.cap; i++ {
		newMessage[i] = mq.message[(mq.head+i)%mq.cap]
	}

	mq.head = 0
	mq.tail = mq.cap
	mq.cap = mq.cap * 2
	mq.message = newMessage
}

func (mq *queue) Push(msg interface{}) {
	mq.lock.Lock()

	mq.message[mq.tail] = msg
	mq.tail++
	if mq.tail >= mq.cap {
		mq.tail = 0
	}

	if mq.head == mq.tail {
		mq.expandQueue()
	}

	mq.lock.Unlock()
}

func (mq *queue) Pushs(msgs []interface{}) {
	mq.lock.Lock()
	var msgLen = len(msgs)
	for i := 0; i < msgLen; i++ {
		mq.message[mq.tail] = msgs[i]
		mq.tail++
		if mq.tail >= mq.cap {
			mq.tail = 0
		}

		if mq.head == mq.tail {
			mq.expandQueue()
		}
	}

	mq.lock.Unlock()
}

func (mq *queue) Pick(retList *[]interface{}) (exit bool) {
	return
}

func (mq *queue) Pops(msgs []interface{}) uint32 {
	var ret uint32
	mq.lock.Lock()
	if mq.head != mq.tail {
		recvLen := len(msgs)
		if recvLen > 0 {
			for i := 0; i < recvLen; i++ {
				msgs[i] = mq.message[mq.head]
				mq.message[mq.head] = nilMsg
				mq.head++
				ret++
				if mq.head >= mq.cap {
					mq.head = 0
				}

				if mq.head == mq.tail {
					break
				}
			}
		}

		var length = mq.tail - mq.head
		if length < 0 {
			length += mq.cap
		}

		for {
			if length > mq.overloadThreshold {
				mq.overload = length
				mq.overloadThreshold *= 2
			} else {
				break
			}
		}
	} else {
		mq.overloadThreshold = messageOverload
		mq.lock.Unlock()
		return 0
	}

	mq.lock.Unlock()

	return ret
}

func (mq *queue) Pop() (interface{}, error) {
	var msg interface{}
	mq.lock.Lock()

	if mq.head != mq.tail {
		msg = mq.message[mq.head]
		mq.message[mq.head] = nilMsg
		mq.head++

		var head = mq.head
		var tail = mq.tail
		var cap = mq.cap

		if head >= cap {
			head = 0
			mq.head = head
		}

		var length = tail - head
		if length < 0 {
			length += cap
		}

		for {
			if length > mq.overloadThreshold {
				mq.overload = length
				mq.overloadThreshold *= 2
			} else {
				break
			}
		}
	} else {
		mq.overloadThreshold = messageOverload
		mq.lock.Unlock()
		return msg, errors.New("message queue length is 0")
	}

	mq.lock.Unlock()

	return msg, nil
}

func (mq *queue) mqOverload() int {
	if mq.overload > 0 {
		var overload = mq.overload
		mq.overload = 0
		return overload
	}

	return 0
}
