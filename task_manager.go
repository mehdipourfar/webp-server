package main

import (
	"fmt"
	"sync"
)

/*
   Main responsiblity of TaskManager is to prevent from
   thundering herd problem in image conversion process.
   When an image is recently uploaded, and multiple users
   request it with the same filters, we should make sure
   that the image conversion process only happens once.
   TaskManager also acts a worker pool and prevents from
   running Convert function in thousands of goroutines.
*/

type ProcessFunc func() error

type Task struct {
	Finished chan struct{}
	Func     *ProcessFunc
	Error    error
}

func (t *Task) Run() {
	defer func() {
		if r := recover(); r != nil {
			t.Error = fmt.Errorf("Task failed: %v", r)
			close(t.Finished)
		}
	}()

	t.Error = (*t.Func)()
	close(t.Finished)
}

type TaskManager struct {
	Tasks   map[string]*Task
	Request chan *Task
	sync.Mutex
}

func NewTaskManager(workersCount int) *TaskManager {
	t := &TaskManager{
		Tasks:   make(map[string]*Task),
		Request: make(chan *Task, 10),
	}
	t.StartWorkers(workersCount)
	return t
}

func (tm *TaskManager) StartWorkers(count int) {
	for i := 0; i < count; i++ {
		go func() {
			for task := range tm.Request {
				task.Run()
			}
		}()
	}
}

func (tm *TaskManager) Delete(taskId string) {
	tm.Lock()
	delete(tm.Tasks, taskId)
	tm.Unlock()
}

func (tm *TaskManager) RunTask(taskId string, f ProcessFunc) error {
	tm.Lock()
	task := tm.Tasks[taskId]
	if task == nil {
		task = &Task{Finished: make(chan struct{}), Func: &f}
		tm.Tasks[taskId] = task
		tm.Unlock()
		tm.Request <- task
		<-task.Finished
		tm.Delete(taskId)
	} else {
		tm.Unlock()
		<-task.Finished
	}
	return task.Error
}
