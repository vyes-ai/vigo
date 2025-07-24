//
// json.go
// Copyright (C) 2025 veypi <i@veypi.com>
// 2025-07-15 16:00
// Distributed under terms of the MIT license.
//

package common

import (
	"strconv"

	"github.com/vyes/vigo"
	"github.com/vyes/vigo/logv"
)

func JsonResponse(x *vigo.X, data any) error {
	return x.JSON(map[string]any{"code": 0, "data": data})
}

func JsonErrorResponse(x *vigo.X, err error) error {
	code := 400
	if e, ok := err.(*vigo.Error); ok {
		code = e.Code
		if code > 999 {
			code, _ = strconv.Atoi(strconv.Itoa(code)[:3])
		}
		x.WriteHeader(code)
		x.JSON(map[string]any{"code": e.Code, "message": e.Message})
		return nil
	}
	if code != 404 {
		logv.WithDeepCaller.Warn().Msg(err.Error())
	}
	x.WriteHeader(code)
	x.JSON(map[string]any{"code": code, "message": err.Error()})
	return nil
}
