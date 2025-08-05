//
// json.go
// Copyright (C) 2025 veypi <i@veypi.com>
// 2025-07-15 16:00
// Distributed under terms of the MIT license.
//

package common

import (
	"fmt"
	"strconv"

	"github.com/vyes-ai/vigo"
)

func JsonResponse(x *vigo.X, data any) error {
	x.WriteHeader(200)
	return x.JSON(data)
}

func JsonErrorResponse(x *vigo.X, err error) error {
	code := 400
	if e, ok := err.(*vigo.Error); ok {
		code = e.Code
		if code > 999 {
			code, _ = strconv.Atoi(strconv.Itoa(code)[:3])
		}
		x.WriteHeader(code)
		x.Write(fmt.Appendf([]byte{}, `{"code":%d,"message":"%s"}`, e.Code, e.Message))
		return nil
	}
	x.WriteHeader(code)
	x.Write(fmt.Appendf([]byte{}, `{"code":%d,"message":"%s"}`, code, err.Error()))
	return nil
}
