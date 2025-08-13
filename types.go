//
// types.go
// Copyright (C) 2025 veypi <i@veypi.com>
//
// Distributed under terms of the MIT license.
//

package vigo

import (
	"net/http"
	"reflect"
)

// map
type M = map[string]any

// slice
type S = []any

// handlers
type FuncX2None = func(*X)
type FuncX2Any = func(*X) any
type FuncX2Err = func(*X) error
type FuncX2AnyErr = func(*X) (any, error)
type FuncAny2None = func(*X, any)
type FuncAny2Any = func(*X, any) any
type FuncAny2Err = func(*X, any) error
type FuncAny2AnyErr = func(*X, any) (any, error)
type FuncDescription = string

type FuncHttp2None = func(http.ResponseWriter, *http.Request)
type FuncHttp2Any = func(http.ResponseWriter, *http.Request) any
type FuncHttp2Err = func(http.ResponseWriter, *http.Request) error
type FuncHttp2AnyErr = func(http.ResponseWriter, *http.Request) (any, error)

type FuncErr = func(*X, error) error

func IgnoreErr(x *X, err error) error {
	return nil
}

type FuncSkipBefore func()

var SkipBefore FuncSkipBefore = func() {}

func DiliverData(x *X, data any) (any, error) {
	return data, nil
}

type FuncStandard[T, U any] func(*X, T) (U, error)

func Standardize[T any, U any](fc FuncStandard[T, U]) func(*X) (any, error) {

	tType := reflect.TypeOf((*T)(nil)).Elem()
	isPtr := tType.Kind() == reflect.Ptr
	var elemType reflect.Type
	if isPtr {
		elemType = tType.Elem()
	}
	return func(x *X) (any, error) {
		var opts T
		var optsPtr any
		if isPtr {
			// T 是指针类型，需要创建指向的类型的实例
			newValue := reflect.New(elemType)
			opts = newValue.Interface().(T)
			optsPtr = opts
		} else {
			// T 不是指针类型，直接传递地址
			optsPtr = &opts
		}
		err := x.Parse(optsPtr)
		if err != nil {
			return nil, err
		}
		return fc(x, opts)
	}
}

var xType = reflect.TypeOf((*X)(nil))

func TryStandardize(fn any) (func(*X) (any, error), bool) {
	fnType := reflect.TypeOf(fn)

	// 快速检查基本条件
	if fnType.Kind() != reflect.Func ||
		fnType.NumIn() != 2 ||
		fnType.NumOut() != 2 {
		return nil, false
	}

	// 检查第一个参数是否是 *X
	if fnType.In(0) != reflect.TypeOf((*X)(nil)) {
		return nil, false
	}

	// 检查第二个返回值是否是 error
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if !fnType.Out(1).Implements(errorType) {
		return nil, false
	}

	// 创建规则化函数
	return createStandardizedFunc(fn, fnType), true
}

// 使用反射创建规则化的函数
func createStandardizedFunc(originalFn any, fnType reflect.Type) func(*X) (any, error) {
	fnValue := reflect.ValueOf(originalFn)

	// T 是第二个参数的类型
	tType := fnType.In(1)

	return func(x *X) (any, error) {
		// 创建 T 类型的实例
		var opts reflect.Value

		if tType.Kind() == reflect.Ptr {
			// 如果 T 是指针类型，创建指向的类型的实例
			opts = reflect.New(tType.Elem())
		} else {
			// 如果 T 不是指针类型，创建零值
			opts = reflect.New(tType)
		}

		// 调用 Parse 方法
		err := x.Parse(opts.Interface())
		if err != nil {
			return nil, err
		}

		// 准备调用原函数的参数
		var args []reflect.Value
		args = append(args, reflect.ValueOf(x))

		if tType.Kind() == reflect.Ptr {
			args = append(args, opts)
		} else {
			args = append(args, opts.Elem())
		}

		// 调用原函数
		results := fnValue.Call(args)

		// 检查 error
		if !results[1].IsNil() {
			return nil, results[1].Interface().(error)
		}

		// 返回结果
		return results[0].Interface(), nil
	}
}
