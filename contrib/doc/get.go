//
// get.go
// Copyright (C) 2025 veypi <i@veypi.com>
//
// Distributed under terms of the MIT license.
//

package doc

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/vyes-ai/vigo"
)

func (d *DocFS) Dir(x *vigo.X, opts *DirOpts) ([]*ItemResponse, error) {
	// 清理路径并添加前缀
	cleanPath := filepath.Clean(opts.Path)
	if cleanPath == "." {
		cleanPath = ""
	}

	fullPath := filepath.Join(d.prefix, cleanPath)

	// 检查路径是否存在以及是文件还是目录
	fileInfo, err := fs.Stat(d.docFS, fullPath)
	if err != nil {
		return nil, ErrFailRead.WithArgs(fullPath, err)
	}

	// 如果是文件，返回文件本身的信息
	if !fileInfo.IsDir() {
		return []*ItemResponse{
			{
				Name:     filepath.Base(cleanPath),
				Filename: strings.TrimPrefix(fullPath, d.prefix),
				IsDir:    false,
			},
		}, nil
	}

	// 如果是目录，获取目录内容
	var result []*ItemResponse

	// 递归收集文件和目录
	err = d.collectEntries(fullPath, cleanPath, opts.Depth, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (d *DocFS) collectEntries(fullPath, relativePath string, depth int, result *[]*ItemResponse) error {
	// 如果深度为0，不继续遍历
	if depth == 0 {
		return nil
	}

	// 读取目录内容
	entries, err := d.docFS.ReadDir(fullPath)
	if err != nil {
		return ErrFailRead.WithArgs(fullPath, err)
	}

	// 遍历目录条目
	for _, entry := range entries {
		childFullPath := filepath.Join(fullPath, entry.Name())
		childRelativePath := filepath.Join(relativePath, entry.Name())

		// 添加当前条目到结果中
		*result = append(*result, &ItemResponse{
			Name:     entry.Name(),
			Filename: strings.TrimPrefix(childFullPath, d.prefix),
			IsDir:    entry.IsDir(),
		})

		// 如果是目录且需要递归（深度 > 1 或 -1）
		if entry.IsDir() && (depth > 1 || depth == -1) {
			newDepth := depth
			if depth > 1 {
				newDepth = depth - 1
			}

			// 递归收集子目录内容
			err := d.collectEntries(childFullPath, childRelativePath, newDepth, result)
			if err != nil {
				// 记录错误但继续处理其他条目
				continue
			}
		}
	}

	return nil
}
