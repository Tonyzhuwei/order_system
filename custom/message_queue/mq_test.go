package message_queue

import (
	"github.com/stretchr/testify/assert"
	"order_system/model"
	"testing"
)

func TestMessageQueue_Enqueue(t *testing.T) {
	mq := NewMessageQueue()
	mq.Enqueue(&model.Order{})
	assert.Equal(t, 1, mq.GetMsgCount())
	mq.Enqueue(&model.Order{})
	assert.Equal(t, 2, mq.GetMsgCount())
}

func TestMessageQueue_Dequeue(t *testing.T) {
	mq := NewMessageQueue()
	mq.Enqueue(&model.Order{})
	mq.Enqueue(&model.Order{})
	_ = mq.Dequeue()
	assert.Equal(t, 1, mq.GetMsgCount())
	_ = mq.Dequeue()
	assert.Equal(t, 0, mq.GetMsgCount())
}
