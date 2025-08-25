//
// xwriter.go
// Copyright (C) 2025 veypi <i@veypi.com>
//
// Distributed under terms of the MIT license.
//

package vigo

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

func (x *X) Header() http.Header {
	return x.writer.Header()
}

func (x *X) WriteHeader(statusCode int) {
	x.writer.WriteHeader(statusCode)
}

func (x *X) Write(p []byte) (n int, err error) {
	return x.writer.Write(p)
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

func (x *X) Embed(fs *embed.FS, fpath string) error {
	file, err := fs.Open(fpath)
	if err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		return err
	}
	contentType := mime.TypeByExtension(filepath.Ext(fpath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	x.Header().Set("Content-Type", contentType)
	x.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	io.Copy(x, file)
	return nil
}

func (x *X) File(path string) error {
	// 打开文件
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}
	contentType := mime.TypeByExtension(filepath.Ext(path))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	x.Header().Set("Content-Type", contentType)
	x.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))
	io.Copy(x, file)
	return nil
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
