/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package queue

// Queue 相当于容量可无限制的channel
// 设置无限buffer的channel(max<=0)
// Enqueue 接口会阻塞直到可以元素放入队列中，阻塞的情况只在队列满的时候才会出现
// Dequeue 接口会阻塞直到队列中有元素返回，阻塞的情况只在队列空的时候才会出现
type Queue[T any] struct {
	l    Deque[T]
	max  int
	done chan struct{}
	in   chan T // use to enqueue
	out  chan T // use to dequeue
}

// NewQueueWithSize max代表队列元素个数上限，若小于等于0，则队列无元素上限
// 内部会启动一个goroutine用于channel同步，可用Destroy()方法销毁。
// 注意调用Destroy()后就不可执行入队出队操作，否则会一直阻塞下去。
func NewQueueWithSize[T any](max int, inChanSize int) *Queue[T] {
	q := &Queue[T]{
		max:  max,
		done: make(chan struct{}),
		in:   make(chan T, inChanSize),
		out:  make(chan T),
	}
	go q.dispatch()
	return q
}

func NewQueue[T any]() *Queue[T] {
	return NewQueueWithSize[T](0, 0)
}

func (q *Queue[T]) dispatch() {
	for {
		if q.l.Len() == 0 {
			// the queue is empty, only enqueue is allowed.
			select {
			case v := <-q.in:
				q.l.PushBack(v)
			case <-q.done:
				return
			}
		}
		e, ok := q.l.TryPopFront()
		if !ok {
			continue
		}

		if q.max > 0 && q.l.Len() >= q.max {
			// the queue is full, only dequeue is allowed.
			select {
			case q.out <- e:
				q.l.Remove(0)
			case <-q.done:
				return
			}
		} else {
			// enqueue and dequeue are allowed.
			select {
			case v := <-q.in:
				q.l.PushBack(v)
			case q.out <- e:
				q.l.Remove(0)
			case <-q.done:
				return
			}
		}
	}
}

func (q *Queue[T]) Size() int {
	return q.l.Len()
}

func (q *Queue[T]) MaxSize() int {
	return q.max
}

func (q *Queue[T]) TuneSize(max int) {
	q.max = max
}

func (q *Queue[T]) Enqueue(v T) {
	q.in <- v
}

func (q *Queue[T]) Dequeue() T {
	return <-q.out
}

func (q *Queue[T]) EnqueueC() chan<- T {
	return q.in
}

func (q *Queue[T]) DequeueC() <-chan T {
	return q.out
}

func (q *Queue[T]) Destroy() {
	// cancel dispatch goroutine
	if q.done != nil {
		// q.done <- struct{}{}
		close(q.done)
	}
}
