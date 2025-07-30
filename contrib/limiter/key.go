//
// key.go
// Copyright (C) 2025 veypi <i@veypi.com>
//
// Distributed under terms of the MIT license.
//

package limiter

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

// getRealIP 获取真实IP地址
func GetRealIP(r *http.Request) string {
	// 检查 X-Forwarded-For 头
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// 检查 X-Real-IP 头
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// 从 RemoteAddr 获取
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// getPathKeyFunc 基于路径的key生成函数
func GetPathKeyFunc(r *http.Request) string {
	return fmt.Sprintf("%s:%s", GetRealIP(r), r.URL.Path)
}
