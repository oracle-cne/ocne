package qcow2

import (
	"container/list"
	"context"
	"sync"
)

type AioTaskFunc func(task *Qcow2Task) error
type AioTaskRoutineFunc func(ctx context.Context, taskList *SignalList)

type Qcow2Task struct {
	bs             *BlockDriverState
	subclusterType QCow2SubclusterType
	hostOffset     uint64
	offset         uint64
	bytes          uint64
	qiov           *QEMUIOVector
	qiovOffset     uint64
	l2meta         *QCowL2Meta /* only for write */
	completeCh     chan any
	errorCh        chan error
	taskFunc       AioTaskFunc
}

type SignalList struct {
	list     *list.List
	awakenCh chan any
	lock     sync.RWMutex
}

func NewSignalList() *SignalList {
	return &SignalList{
		list:     list.New(),
		awakenCh: make(chan any),
	}
}

func (l *SignalList) Push(ele any) {
	l.lock.Lock()
	l.list.PushBack(ele)
	l.lock.Unlock()
	l.awakenCh <- struct{}{}
}

func (l *SignalList) Pop() any {
	l.lock.Lock()
	defer l.lock.Unlock()
	ele := l.list.Front()
	if ele != nil {
		l.list.Remove(ele)
		return ele.Value
	}
	return nil
}

func qcow2_aio_routine(ctx context.Context, taskList *SignalList) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-taskList.awakenCh:
			for {
				obj := taskList.Pop()
				if obj == nil {
					break
				}
				task := obj.(*Qcow2Task)
				if err := task.taskFunc(task); err != nil {
					task.errorCh <- err
				} else {
					task.completeCh <- struct{}{}
				}
			}
		}
	}
}
