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

	"github.com/vyes-ai/vigo"
	"github.com/vyes-ai/vigo/contrib/common"
	"github.com/vyes-ai/vigo/utils"
)

func WrapUI(router vigo.Router, uiFS embed.FS, args ...string) vigo.Router {
	current := utils.CurrentDir(1)
	vdev := os.Getenv("vdev")
	renderEnv := func(x *vigo.X) {
		x.Header().Set("vyes-root", router.String())
		x.Header().Set("vyes-vdev", vdev)
		for i := 0; i < len(args); i += 2 {
			x.Header().Set("vyes-"+args[i], args[i+1])
		}
	}
	if vdev != "" && current != "" {
		router.Get("/*path", renderEnv, common.Static(path.Join(current, "ui"), "root.html"))
	} else {
		router.Get("/*path", renderEnv, common.EmbedDir(uiFS, "ui", "root.html"))
	}
	return router
}
