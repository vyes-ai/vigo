//
// vyes.go
// Copyright (C) 2025 veypi <i@veypi.com>
// 2025-04-01 18:33
// Distributed under terms of the MIT license.
//

package vyes

import (
	"embed"
	"os"
	"path"

	"github.com/vyes/vigo"
	"github.com/vyes/vigo/middlewares/common"
	"github.com/vyes/vigo/utils"
)

func WrapUI(router vigo.Router, uiFS embed.FS) bool {
	current := utils.CurrentDir(1)
	vdev := os.Getenv("vdev")
	var r vigo.Router
	res := false
	if vdev != "" && current != "" {
		res = true
		r = router.Get("/*path", common.Static(path.Join(current, "ui"), "root.html"))
	} else {
		r = router.Get("/*path", common.EmbedDir(uiFS, "ui", "root.html"))
	}
	r.UseBefore(func(x *vigo.X) {
		x.Header().Set("vyes-root", router.String())
		x.Header().Set("vyes-vdev", vdev)
	})
	return res
}
