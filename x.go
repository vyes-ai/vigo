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
	"runtime/debug"
	"strconv"
	"strings"
	"sync"

	"github.com/vyes/vigo/logv"
)

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
	defer func() {
		if e := recover(); e != nil {
			if e2, ok := e.(error); ok {
				err = e2
			} else {
				err = fmt.Errorf("%s: %v", ErrCrash, e)
			}
			x.handleErr(err)
			logv.Warn().Msgf("%s", debug.Stack())
		}
	}()
	if x.fid >= len(x.fcs) {
		return
	}
	fc := x.fcs[x.fid]
	x.fid++
	var arg any
	switch fc := fc.(type) {
	case FuncX2None:
		fc(x)
	case FuncX2Any:
		arg = fc(x)
	case FuncX2Err:
		err = fc(x)
	case FuncX2AnyErr:
		arg, err = fc(x)
	case FuncAny2None:
		fc(x, args[0])
	case FuncAny2Any:
		arg = fc(x, args[0])
	case FuncAny2Err:
		err = fc(x, args[0])
	case FuncAny2AnyErr:
		arg, err = fc(x, args[0])
	case FuncHttp2None:
		fc(x.ResponseWriter(), x.Request)
	case FuncHttp2Any:
		arg = fc(x.ResponseWriter(), x.Request)
	case FuncHttp2Err:
		err = fc(x.ResponseWriter(), x.Request)
	case FuncHttp2AnyErr:
		arg, err = fc(x.ResponseWriter(), x.Request)
	case FuncErr:
		// do nothing
	case FuncDescription:
	default:
		logv.Warn().Msgf("unknown func type %T", fc)
	}
	if err != nil {
		x.handleErr(err)
		return
	}
	x.Next(arg)
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
				// x.Next()
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
func (x *X) SSEEvent() func(event string, data any) {
	x.writer.Header().Set("Content-Type", "text/event-stream")
	x.writer.Header().Set("Cache-Control", "no-cache")
	x.writer.Header().Set("Connection", "keep-alive")
	return func(event string, data any) {
		if event != "" {
			fmt.Fprintf(x.writer, "event: %s\n", event)
		}
		if data != nil {
			dataStr, ok := data.(string)
			if !ok {
				dataStr = fmt.Sprintf("%v", data)
			}
			fmt.Fprintf(x.writer, "data: %s\n\n", dataStr)
		} else {
			fmt.Fprint(x.writer, "\n")
		}
		f := x.writer.(http.Flusher)
		f.Flush()
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
