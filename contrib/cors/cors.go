//
// cors.go
// Copyright (C) 2024 veypi <i@veypi.com>
// 2024-12-06 15:56
// Distributed under terms of the MIT license.
//

package cors

import (
	"net/http"
	"slices"

	"github.com/vyes/vigo"
)

func AllowAny(x *vigo.X) {
	origin := x.Request.Header.Get("Origin")
	x.Header().Set("Access-Control-Allow-Origin", origin)
	x.Header().Set("Access-Control-Allow-Credentials", "true")
	x.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH, PROPFIND")
	x.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, depth")
	if x.Request.Method == http.MethodOptions && x.Request.Header.Get("Access-Control-Request-Method") != "" {
		x.Stop()
	}
}

func CorsAllow(domains ...string) func(x *vigo.X) {
	return func(x *vigo.X) {
		origin := x.Request.Header.Get("Origin")
		if slices.Contains(domains, origin) {
			x.Header().Set("Access-Control-Allow-Origin", origin)
			x.Header().Set("Access-Control-Allow-Credentials", "true")
			x.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
			x.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
			if x.Request.Method == http.MethodOptions && x.Request.Header.Get("Access-Control-Request-Method") != "" {
				x.Stop()
			}
			return
		}
	}
}
