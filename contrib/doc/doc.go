package doc

import (
	"embed"

	"github.com/vyes-ai/vigo"
	"github.com/vyes-ai/vigo/contrib/common"
)

var (
	ErrFailRead = vigo.NewError("failed to read file or directory: %s\n%s").WithCode(500)
)

func New(r vigo.Router, docFS embed.FS, prefix string) *DocFS {
	d := &DocFS{
		docFS:  docFS,
		prefix: prefix,
		router: r,
	}
	d.router.UseAfter(common.JsonResponse, common.JsonErrorResponse)
	// d.router.Get("/", vigo.Standardize(d.Dir))
	d.router.Post("/", vigo.Standardize(d.List))
	return d
}

type DocFS struct {
	docFS  embed.FS
	prefix string
	router vigo.Router
}

type ItemResponse struct {
	Name     string `json:"name"`
	Filename string `json:"filename" usage:"absolute path"`
	IsDir    bool   `json:"is_dir"`
}

type DirOpts struct {
	Path  string `json:"path" parse:"query" usage:"The path to the directory."`
	Depth int    `json:"depth" parse:"query" usage:"The depth of the directory to list. -1 is unlimit" default:"1"`
}

type ListOpts struct {
	Path  string `json:"path" usage:"The prefix path."`
	Depth int    `json:"depth" usage:"The depth of the directory to list. -1 is unlimit" default:"0"`
}
