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
	Signal chan struct{}
	Error  error
}

type TaskStorage struct {
	Tasks map[string]*Task
	sync.Mutex
}

func NewTaskStorage() *TaskStorage {
	return &TaskStorage{Tasks: make(map[string]*Task)}
}

func (task *Task) Done(err error) {
	task.Error = err
	close(task.Signal)
}

func (taskStorage *TaskStorage) Delete(taskId string) {
	taskStorage.Lock()
	delete(taskStorage.Tasks, taskId)
	taskStorage.Unlock()
}

type ProcessFunc func() error

func (taskStorage *TaskStorage) Run(taskId string, f ProcessFunc) error {
	taskStorage.Lock()
	task := taskStorage.Tasks[taskId]
	if task == nil {
		task = &Task{Signal: make(chan struct{})}
		taskStorage.Tasks[taskId] = task
		taskStorage.Unlock()
		func() {
			defer func() {
				// panic handling for ProcessFunc.
				// This defered function ensures that the task will be cleared
				// and channel will be closed even in case of a panic.

				if r := recover(); r != nil {
					task.Done(fmt.Errorf("Task failed: %v", r))
					taskStorage.Delete(taskId)
				}
			}()
			task.Done(f())
			taskStorage.Delete(taskId)
		}()
	} else {
		taskStorage.Unlock()
		<-task.Signal
	}
	return task.Error
}
