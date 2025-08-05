//
// x.go
// Copyright (C) 2024 veypi <i@veypi.com>
// 2024-08-09 13:08
// Distributed under terms of the MIT license.
//

package vigo

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"

	"github.com/vyes-ai/vigo/logv"
)

const version = "v0.5.0"

type X struct {
	writer  http.ResponseWriter
	Request *http.Request
	Params  Params
	fcs     []any
	fid     int
}

var _ http.ResponseWriter = &X{}

func (x *X) Stop() {
	x.fid = 99999999
}

func (x *X) Skip(counts ...uint) {
	count := 1
	if len(counts) > 0 {
		count = int(counts[0])
	}
	x.fid += int(count)
}

func (x *X) Next(args ...any) {
	// args[0] vaild
	var err error
	var response any
	defer func() {
		if e := recover(); e != nil {
			if e2, ok := e.(error); ok {
				err = e2
			} else {
				err = fmt.Errorf("%s: %v", ErrCrash, e)
			}
			if ve, ok := err.(*Error); ok {
				// 有特别明确需求取调用panic(vigo.Error)不打印堆栈
				logv.WithNoCaller.Warn().Msgf("panic: %s, code: %d", ve.Message, ve.Code)
			} else {
				logv.WithNoCaller.Error().Msgf("%s", debug.Stack())
			}
			x.handleErr(err)
		}
	}()
	if x.fid >= len(x.fcs) {
		return
	}
	fc := x.fcs[x.fid]
	x.fid++
	var arg any
	if len(args) > 0 {
		arg = args[0]
	}
	switch fc := fc.(type) {
	case FuncX2None:
		fc(x)
	case FuncX2Any:
		response = fc(x)
	case FuncX2Err:
		err = fc(x)
	case FuncX2AnyErr:
		response, err = fc(x)
	case FuncAny2None:
		fc(x, arg)
	case FuncAny2Any:
		response = fc(x, arg)
	case FuncAny2Err:
		err = fc(x, arg)
	case FuncAny2AnyErr:
		response, err = fc(x, arg)
	case FuncHttp2None:
		fc(x.ResponseWriter(), x.Request)
	case FuncHttp2Any:
		response = fc(x.ResponseWriter(), x.Request)
	case FuncHttp2Err:
		err = fc(x.ResponseWriter(), x.Request)
	case FuncHttp2AnyErr:
		response, err = fc(x.ResponseWriter(), x.Request)
	case FuncErr:
		// do nothing
	case FuncDescription:
	default:
		logv.Warn().Msgf("unknown func type %T", fc)
	}
	if err != nil {
		logv.WithNoCaller.Info().Msgf("%s return error: %v", runtime.FuncForPC(reflect.ValueOf(fc).Pointer()).Name(), err)
		x.handleErr(err)
		return
	}
	x.Next(response)
}

func (x *X) handleErr(err error) bool {
	if x.fid >= len(x.fcs) {
		logv.Warn().Msgf("unhandled error: %v", err)
		return false
	}
	for x.fid < len(x.fcs) {
		fc, ok := x.fcs[x.fid].(FuncErr)
		x.fid++
		if ok {
			err = fc(x, err)
			if err == nil {
				return true
			}
		}
	}
	logv.Warn().Msgf("unhandled error: %v", err)
	return false
}

func (x *X) ResponseWriter() http.ResponseWriter {
	return x.writer
}

func (x *X) Get(key string) any {
	return x.Request.Context().Value(key)
}

func (x *X) Set(key string, value any) {
	if x.Request == nil {
		logv.Warn().Msgf("set %s=%v to nil request", key, value)
		return
	}
	x.Request = x.Request.WithContext(context.WithValue(x.Request.Context(), key, value))
}

func (x *X) Write(p []byte) (n int, err error) {
	return x.writer.Write(p)
}

func (x *X) WriteHeader(statusCode int) {
	x.writer.WriteHeader(statusCode)
}
func (x *X) Header() http.Header {
	return x.writer.Header()
}

func (x *X) JSON(data any) error {
	var err error
	switch v := data.(type) {
	case string:
		_, err = x.writer.Write([]byte(v))
	case []byte:
		_, err = x.writer.Write(v)
	case error:
		_, err = x.writer.Write([]byte(v.Error()))
	case nil:
	case int, uint, int8, uint8, int16, uint16, int32, uint32, int64, uint64, float32, float64, bool:
		_, err = x.writer.Write((fmt.Appendf([]byte{}, "%v", v)))
	default:
		b, err := json.Marshal(data)
		if err != nil {
			return err
		}
		x.Header().Add("Content-Type", "application/json")
		_, err = x.Write(b)
	}
	return err
}

func (x *X) SSEWriter() func(p []byte) (int, error) {
	x.writer.Header().Set("Content-Type", "text/event-stream")
	x.writer.Header().Set("Cache-Control", "no-cache")
	x.writer.Header().Set("Connection", "keep-alive")
	f := x.writer.(http.Flusher)
	fc := func(p []byte) (int, error) {
		l, err := x.writer.Write(p)
		if err != nil {
			return l, err
		}
		f.Flush()
		return l, nil
	}
	return fc
}

func (x *X) SSEEvent() func(string, any) (int, error) {
	x.writer.Header().Set("Content-Type", "text/event-stream")
	x.writer.Header().Set("Cache-Control", "no-cache")
	x.writer.Header().Set("Connection", "keep-alive")
	return func(event string, data any) (n int, err error) {
		if event != "" && event != "data" {
			if nn, err := fmt.Fprintf(x.writer, "event: %s\n", event); err != nil {
				return nn, err
			} else {
				n = n + nn
			}
		}
		if data != nil {
			if nn, err := fmt.Fprintf(x.writer, "data: %s\n\n", data); err != nil {
				return nn + n, err
			} else {
				n = n + nn
			}
		} else {
			if nn, err := fmt.Fprint(x.writer, "\n"); err != nil {
				return nn + n, err
			} else {
				n = n + nn
			}
		}
		f := x.writer.(http.Flusher)
		f.Flush()
		return n, err
	}
}

func (x *X) Context() context.Context {
	return x.Request.Context()
}

func (x *X) GetRemoteIp() string {
	// 首先尝试从 X-Forwarded-For 获取 IP 地址
	ip := x.Request.Header.Get("X-Forwarded-For")
	if ip != "" {
		// X-Forwarded-For 可能包含多个 IP 地址，以逗号分隔，
		// 这里我们取第一个 IP 地址作为客户端的 IP。
		return strings.TrimSpace(strings.Split(ip, ",")[0])
	}

	// 如果 X-Forwarded-For 不存在，则尝试从 X-Real-IP 获取 IP 地址
	ip = x.Request.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	// 如果以上两个都没有，则直接从 RemoteAddr 获取 IP 地址
	ip, _, err := net.SplitHostPort(x.Request.RemoteAddr)
	if err != nil {
		return ""
	}
	return ip
}

func (x *X) setParam(k string, v string) {
	for _, p := range x.Params {
		if p[0] == k {
			p[1] = v
			return
		}
	}
	x.Params = append(x.Params, [2]string{k, v})
}

type Params [][2]string

func (ps *Params) Try(key string) (string, bool) {
	for _, p := range *ps {
		if key == p[0] {
			return p[1], true
		}
	}
	return "", false
}

func (ps *Params) Get(key string) string {
	v, _ := ps.Try(key)
	return v
}

func (ps *Params) GetInt(k string) int {
	v, _ := ps.Try(k)
	vv, _ := strconv.Atoi(v)
	return vv
}

var xPool = sync.Pool{
	New: func() any {
		return &X{
			Params: make(Params, 0),
		}
	},
}

func acquire() *X {
	x := xPool.Get().(*X)
	return x
}

func release(x *X) {
	x.fid = 0
	x.Params = x.Params[0:0]
	x.Request = nil
	x.writer = nil
	x.fcs = nil
	xPool.Put(x)
}
