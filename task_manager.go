package main

import (
	"fmt"
	"sync"
)

//ProcessFunc is the function responsible for handling task
type ProcessFunc func() error

type task struct {
	finished chan struct{}
	function *ProcessFunc
	err      error
}

func (t *task) run() {
	defer func() {
		if r := recover(); r != nil {
			t.err = fmt.Errorf("Task failed: %v", r)
			close(t.finished)
		}
	}()

	t.err = (*t.function)()
	close(t.finished)
}

//TaskManager is responsible for preventing
//thundering herd problem in image conversion process.
//When an image is recently uploaded, and multiple users
//request it with the same filters, we should make sure
//that the image conversion process only happens once.
//TaskManager also acts a worker pool and prevents from
//running Convert function in thousands of goroutines.
type TaskManager struct {
	tasks   map[string]*task
	request chan *task
	sync.Mutex
}

//NewTaskManager takes the number of background workers
//and creates a new Task manager with spawned workers
func NewTaskManager(workersCount int) *TaskManager {
	t := &TaskManager{
		tasks:   make(map[string]*task),
		request: make(chan *task, 10),
	}
	t.startWorkers(workersCount)
	return t
}

func (tm *TaskManager) startWorkers(count int) {
	for i := 0; i < count; i++ {
		go func() {
			for t := range tm.request {
				t.run()
			}
		}()
	}
}

func (tm *TaskManager) clear(taskID string) {
	tm.Lock()
	delete(tm.tasks, taskID)
	tm.Unlock()
}

//RunTask takes a uniqe taskID and a processing function
//and runs the function in the background
func (tm *TaskManager) RunTask(taskID string, f ProcessFunc) error {
	tm.Lock()
	t := tm.tasks[taskID]
	if t == nil {
		// similar task does not exist at the moment
		t = &task{finished: make(chan struct{}), function: &f}
		tm.tasks[taskID] = t
		tm.Unlock()
		tm.request <- t
		<-t.finished
		tm.clear(taskID)
	} else {
		// task is being done by another process
		tm.Unlock()
		<-t.finished
	}
	return t.err
}
