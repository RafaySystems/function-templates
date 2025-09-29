package sdk

import (
	"errors"
	"fmt"
	"runtime"
)

const (
	ErrCodeUnspecified errorCode = iota
	ErrCodeExecuteAgain
	ErrCodeFailed
	ErrCodeTransient
	ErrCodeNotFound
	ErrCodeConflict
)

type errorCode int

type ErrFunction struct {
	ErrCode    errorCode      `json:"error_code"`
	Message    string         `json:"message"`
	StackTrace []stackFrame   `json:"stack_trace"`
	Data       map[string]any `json:"data"`
}

type stackFrame struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Function string `json:"function"`
}

func (e *ErrFunction) Error() string {
	return fmt.Sprintf("error_code: %d, message: %s", e.ErrCode, e.Message)
}

type errExecuteAgain struct {
	Message string
	Data    map[string]any
}

func (e *errExecuteAgain) Error() string {
	return e.Message
}

func (e *errExecuteAgain) Is(err error) bool {
	_, ok := err.(*errExecuteAgain)
	return ok
}

func (e *errExecuteAgain) Unwrap() error {
	return &ErrFunction{Message: e.Message, ErrCode: ErrCodeExecuteAgain, Data: e.Data}
}

type errFailed struct {
	Message    string
	StackTrace []stackFrame
}

func (e *errFailed) Error() string {
	return e.Message
}

func (e *errFailed) Is(err error) bool {
	_, ok := err.(*errFailed)
	return ok
}

func (e *errFailed) Unwrap() error {
	return &ErrFunction{Message: e.Message, ErrCode: ErrCodeFailed, StackTrace: e.StackTrace}
}

type errTransient struct {
	Message string
}

func (e *errTransient) Error() string {
	return e.Message
}

func (e *errTransient) Is(err error) bool {
	_, ok := err.(*errTransient)
	return ok
}

func (e *errTransient) Unwrap() error {
	return &ErrFunction{Message: e.Message, ErrCode: ErrCodeTransient}
}

type errNotFound struct {
	Message string
}

func (e *errNotFound) Error() string {
	return e.Message
}

func (e *errNotFound) Is(err error) bool {
	_, ok := err.(*errNotFound)
	return ok
}

func (e *errNotFound) Unwrap() error {
	return &ErrFunction{Message: e.Message, ErrCode: ErrCodeNotFound}
}

func IsErrExecuteAgain(err error) bool {
	return errors.Is(err, &errExecuteAgain{})
}

func IsErrFailed(err error) bool {
	return errors.Is(err, &errFailed{})
}

func IsErrTransient(err error) bool {
	return errors.Is(err, &errTransient{})
}

func IsErrFunction(err error) bool {
	return errors.Is(err, &ErrFunction{})
}

func IsErrNotFound(err error) bool {
	return errors.Is(err, &errNotFound{})
}

func AsErrFunction(err error) (*ErrFunction, bool) {
	var ef *ErrFunction
	ok := errors.As(err, &ef)
	if ok {
		// preserve last error message
		ef.Message = err.Error()
	}
	return ef, ok
}

func NewErrExecuteAgain(msg string, data map[string]any) error {
	return &errExecuteAgain{Message: msg, Data: data}
}

func NewErrFailed(msg string) error {
	return &errFailed{Message: msg}
}

func newErrFailedWithStackTrace(msg string) error {
	pc := make([]uintptr, 32)
	n := runtime.Callers(3, pc)
	if n == 0 {
		return &errFailed{Message: msg}
	}
	pc = pc[:n]
	frames := runtime.CallersFrames(pc)
	stack := make([]stackFrame, 0, n)
	for {
		frame, more := frames.Next()
		stack = append(stack, stackFrame{
			File:     frame.File,
			Line:     frame.Line,
			Function: frame.Function,
		})
		if !more {
			break
		}
	}

	return &errFailed{Message: msg, StackTrace: stack}
}

func NewErrTransient(msg string) error {
	return &errTransient{Message: msg}
}

func NewErrNotFound(msg string) error {
	return &errNotFound{Message: msg}
}

type errConflict struct {
	Message string
}

func (e *errConflict) Error() string {
	return e.Message
}

func (e *errConflict) Is(err error) bool {
	_, ok := err.(*errConflict)
	return ok
}

func (e *errConflict) Unwrap() error {
	return &ErrFunction{Message: e.Message, ErrCode: ErrCodeConflict}
}

func IsErrConflict(err error) bool {
	return errors.Is(err, &errConflict{})
}

func NewErrConflict(msg string) error {
	return &errConflict{Message: msg}
}
