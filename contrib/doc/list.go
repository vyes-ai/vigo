//
// list.go
// Copyright (C) 2025 veypi <i@veypi.com>
//
// Distributed under terms of the MIT license.
//

package doc

import (
	"path/filepath"
	"strings"

	"github.com/vyes-ai/vigo"
	"github.com/vyes-ai/vigo/logv"
)

func (d *DocFS) List(x *vigo.X, opts *ListOpts) ([]*ItemResponse, error) {

	// 清理路径
	searchPrefix := filepath.Clean(opts.Path)
	if searchPrefix == "." {
		searchPrefix = ""
	}

	result := make([]*ItemResponse, 0)

	// 从根目录开始搜索
	err := d.searchByPrefix(d.prefix, "", searchPrefix, opts.Depth, 0, &result)
	logv.Warn().Msgf("%s: %v", searchPrefix, err)
	if err != nil {
		return nil, err
	}
	logv.Warn().Msgf("%+v %v", opts, result)
	return result, nil
}

func (d *DocFS) searchByPrefix(currentPath, relativePath, searchPrefix string, maxDepth, currentDepth int, result *[]*ItemResponse) error {
	// 如果设置了深度限制且已达到限制，停止搜索
	if maxDepth >= 0 && currentDepth > maxDepth {
		return nil
	}

	// 读取当前目录内容
	entries, err := d.docFS.ReadDir(currentPath)
	if err != nil {
		return ErrFailRead.WithArgs(currentPath, err)
	}

	for _, entry := range entries {
		entryPath := filepath.Join(currentPath, entry.Name())
		entryRelativePath := filepath.Join(relativePath, entry.Name())

		// 如果是文件，检查是否匹配前缀
		if !entry.IsDir() {
			if d.matchesPrefix(entryRelativePath, searchPrefix) {
				*result = append(*result, &ItemResponse{
					Name:     entry.Name(),
					Filename: strings.TrimPrefix(entryPath, d.prefix),
					IsDir:    false,
				})
			}
		} else {
			// 如果是目录，检查是否需要继续递归搜索
			// 1. 如果目录名本身匹配前缀，需要搜索其内容
			// 2. 如果搜索前缀可能在此目录下，也需要搜索
			if d.shouldSearchDirectory(entryRelativePath, searchPrefix) {
				err := d.searchByPrefix(entryPath, entryRelativePath, searchPrefix, maxDepth, currentDepth+1, result)
				if err != nil {
					// 记录错误但继续处理其他条目
					continue
				}
			}
		}
	}

	return nil
}

// 检查文件路径是否匹配搜索前缀
func (d *DocFS) matchesPrefix(filePath, searchPrefix string) bool {
	// 移除开头的斜杠进行比较
	cleanFilePath := strings.TrimPrefix(filePath, "/")
	cleanSearchPrefix := strings.TrimPrefix(searchPrefix, "/")

	if cleanSearchPrefix == "" {
		return true // 空前缀匹配所有文件
	}

	return strings.HasPrefix(cleanFilePath, cleanSearchPrefix)
}

// 检查是否应该搜索该目录
func (d *DocFS) shouldSearchDirectory(dirPath, searchPrefix string) bool {
	cleanDirPath := strings.TrimPrefix(dirPath, "/")
	cleanSearchPrefix := strings.TrimPrefix(searchPrefix, "/")

	if cleanSearchPrefix == "" {
		return true // 空前缀搜索所有目录
	}

	// 如果搜索前缀以目录路径开头，需要搜索该目录
	// 例如：搜索 "abc/def"，当前目录是 "abc"，需要继续搜索
	return strings.HasPrefix(cleanSearchPrefix, cleanDirPath+"/") ||
		strings.HasPrefix(cleanDirPath, cleanSearchPrefix) ||
		cleanDirPath == ""
}
