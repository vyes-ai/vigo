//
// models.go
// Copyright (C) 2025 veypi <i@veypi.com>
// 2025-05-26 15:19
// Distributed under terms of the MIT license.
//

package dbmodels

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"gorm.io/gorm"
)

type ModelList struct {
	list []any
}

func (ms *ModelList) Add(model any) {
	ms.list = append(ms.list, model)
}
func (ms *ModelList) GetList() []any {
	return ms.list
}

func (ms *ModelList) Append(models ...any) {
	ms.list = append(ms.list, models...)
}

func (ms *ModelList) AutoMigrate(db *gorm.DB) error {
	items := make([]any, 0, 10)
	for _, obj := range ms.list {
		items = append(items, obj)
	}
	db.DisableForeignKeyConstraintWhenMigrating = true
	err := db.AutoMigrate(items...)
	if err != nil {
		return err
	}
	db.DisableForeignKeyConstraintWhenMigrating = false
	return db.AutoMigrate(items...)
}

func (ms *ModelList) AutoDrop(db *gorm.DB) error {
	fmt.Print("\ncontinue to drop db？(yes/no): ")
	// 读取用户输入
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "yes" || input == "y" {
		items := make([]any, 0, 10)
		for _, obj := range ms.list {
			items = append(items, obj)
		}
		return db.Migrator().DropTable(items...)
	}
	return nil
}
