package tasks

import (
	"context"
	"errors"
	"fmt"
	"github.com/opentracing/opentracing-go"
	"reflect"
)

var ErrTaskPanicked = errors.New("Invoking task caused a panic")

type Task struct {
	TaskFunc   reflect.Value
	UseContext bool
	Context    context.Context
	Args       []reflect.Value
}

func New(taskFunc interface{}, args [] Arg) (*Task, error) {
	task := &Task{
		TaskFunc: reflect.ValueOf(taskFunc),
		Context:  context.Background(),
	}

	if err := task.ReflectArgs(args); err != nil {
		fmt.Println("------error-------")
		return nil, fmt.Errorf("lz Reflect task args error: %s", err)
	}

	return task, nil
}

// func remote call
func (task *Task) Call() ([]*TaskResult, error) {

	if span := opentracing.SpanFromContext(task.Context); span != nil {
		defer span.Finish()
	}

	defer func() { //日志输出打印

	}()

	args := task.Args
	//if task.UseContext {
	//	ctxValue := reflect.ValueOf(task.Context)
	//	args = append([]reflect.Value{ctxValue}, args...)
	//}

	results := task.TaskFunc.Call(args)
	if len(results) == 0 {
		return nil, ErrTaskReturnNoValue
	}

	lastResult := results[len(results)-1]

	if !lastResult.IsNil() {
		retryErrorInterface := reflect.TypeOf((*Retriable)(nil)).Elem()
		if lastResult.Type().Implements(retryErrorInterface) {
			return nil, lastResult.Interface().(ErrRetryTaskLater)
		}

		//cheeck thee last return value
		errorInterface := reflect.TypeOf((*error)(nil)).Elem()
		if !lastResult.Type().Implements(errorInterface) {
			return nil, ErrLastReturnError
		}

		return nil, lastResult.Interface().(error)
	}

	taskResults := make([]*TaskResult, len(results)-1)
	for i := 0; i < len(results)-1; i++ {
		val := results[i].Interface()
		typeStr := reflect.TypeOf(val).String()
		taskResults[i] = &TaskResult{
			Type:  typeStr,
			Value: val,
		}
	}
	return taskResults, nil
}

func (task *Task) ReflectArgs(args [] Arg) error {
	argValues := make([]reflect.Value, len(args))
	for i, arg := range args {
		argValue, err := ReflectValue(arg.Type, arg.Value)
		if err != nil {
			return err
		}
		argValues[i] = argValue
	}
	task.Args = argValues
	return nil
}
