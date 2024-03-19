package sdk

import (
	"errors"
	"fmt"
	"runtime"
)

const (
	errCodeUnspecified errorCode = iota
	errCodeExecuteAgain
	errCodeFailed
	errCodeTransient
)

type errorCode int

type ErrFunction struct {
	ErrCode    errorCode    `json:"errorCode"`
	Message    string       `json:"message"`
	StackTrace []StackFrame `json:"stackTrace"`
}

type StackFrame struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Function string `json:"function"`
}

func (e *ErrFunction) Error() string {
	return e.Message
}

func newErrFunction(code errorCode, msg string) error {
	return &ErrFunction{Message: msg, ErrCode: code}
}

func newErrFunctionWithStackTrace(code errorCode, msg string) error {
	pc := make([]uintptr, 32)
	n := runtime.Callers(3, pc)
	if n == 0 {
		return &ErrFunction{Message: msg, ErrCode: code}
	}
	pc = pc[:n]
	frames := runtime.CallersFrames(pc)
	stack := make([]StackFrame, 0, n)
	for {
		frame, more := frames.Next()
		stack = append(stack, StackFrame{
			File:     frame.File,
			Line:     frame.Line,
			Function: frame.Function,
		})
		if !more {
			break
		}
	}

	return &ErrFunction{Message: msg, ErrCode: code, StackTrace: stack}
}

var ErrExecuteAgain = errors.New("function: execute again")
var ErrFailed = errors.New("function: failed")
var ErrTransient = errors.New("function: transient error")

func IsErrExecuteAgain(err error) bool {
	return errors.Is(err, ErrExecuteAgain)
}

func IsErrFailed(err error) bool {
	return errors.Is(err, ErrFailed)
}

func IsErrTransient(err error) bool {
	return errors.Is(err, ErrTransient)
}

func NewErrExecuteAgain(msg string) error {
	return fmt.Errorf("%w: %s", ErrExecuteAgain, msg)
}

func NewErrFailed(msg string) error {
	return fmt.Errorf("%w: %s", ErrFailed, msg)
}

func NewErrTransient(msg string) error {
	return fmt.Errorf("%w: %s", ErrTransient, msg)
}
