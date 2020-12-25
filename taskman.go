package main

import (
	"fmt"
	"sync"
)

/*
   This code is specifically created for prevention of
   thundering herd problem in image conversion process.
   When an image is recently uploaded and multiple users
   request for it with the same filters, we should make
   sure that image conversion process only happens once.
*/

type Task struct {
	done chan struct{}
	err  error
}

type TaskMan struct {
	Tasks map[string]*Task
	sync.Mutex
}

func NewTaskMan() *TaskMan {
	return &TaskMan{Tasks: make(map[string]*Task)}
}

type ProcessFunc func() error

func (taskMan *TaskMan) Run(uid string, f ProcessFunc) error {
	taskMan.Lock()
	task := taskMan.Tasks[uid]
	if task == nil {
		task = &Task{done: make(chan struct{})}
		taskMan.Tasks[uid] = task
		taskMan.Unlock()
		func() {
			defer func() {
				if r := recover(); r != nil {
					task.err = fmt.Errorf("Task failed: %v", r)
					close(task.done)
				}
			}()
			task.err = f()
			close(task.done)
		}()
	} else {
		taskMan.Unlock()
		<-task.done
		taskMan.Lock()
		delete(taskMan.Tasks, uid)
		taskMan.Unlock()
	}
	return task.err
}
