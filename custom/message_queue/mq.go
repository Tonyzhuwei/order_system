package message_queue

import (
	"order_system/model"
)

type MessageQueue struct {
	channel chan *model.Order
}

// NewMessageQueue A lightweight message queue based on Golang channel, not support message persistence.
func NewMessageQueue() *MessageQueue {
	newChan := make(chan *model.Order, 10000)
	return &MessageQueue{
		channel: newChan,
	}
}

func (mq *MessageQueue) Enqueue(msg *model.Order) {
	mq.channel <- msg
}

func (mq *MessageQueue) Dequeue() *model.Order {
	return <-mq.channel
}

func (mq *MessageQueue) GetMsgCount() int {
	return len(mq.channel)
}

func (mq *MessageQueue) CloseQueue() {
	close(mq.channel)
}
