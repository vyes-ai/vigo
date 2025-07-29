//
// register.go
// Copyright (C) 2025 veypi <i@veypi.com>
//
// Distributed under terms of the MIT license.
//

package flags

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/vyes/vigo/logv"
)

// isZeroValue 检查值是否为零值
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	default:
		// 对于其他复杂类型，认为不是零值
		return false
	}
}

// getDefaultValue 获取字段的默认值（优先使用字段当前值，其次使用default标签）
func getDefaultValue(field reflect.Value, defaultTag string) string {
	// 如果字段值不是零值，则使用字段的当前值作为默认值
	if !isZeroValue(field) {
		switch field.Kind() {
		case reflect.String:
			return field.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return strconv.FormatInt(field.Int(), 10)
		case reflect.Bool:
			return strconv.FormatBool(field.Bool())
		case reflect.Float32, reflect.Float64:
			return strconv.FormatFloat(field.Float(), 'g', -1, 64)
		}
	}

	// 否则使用default标签的值
	return defaultTag
}

// AutoRegister 自动注册命令行参数
func (fs *Flags) AutoRegister(config any) {
	v := reflect.ValueOf(config)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		logv.Warn().Msgf("config must be a pointer to a struct, got %T", config)
		return
	}

	v = v.Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// 跳过未导出的字段
		if !field.CanSet() {
			continue
		}

		// 获取 json tag 作为参数名
		jsonTag := fieldType.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue // 如果没有 json tag 或者是空字符串，则跳过
		} else if strings.Contains(jsonTag, ",") {
			jsonTag = strings.Split(jsonTag, ",")[0]
		}

		// 获取 default tag
		defaultTag := fieldType.Tag.Get("default")

		// 获取实际的默认值（优先使用字段当前值）
		defaultValue := getDefaultValue(field, defaultTag)

		// 获取字段的描述
		usage := fieldType.Tag.Get("usage")
		if usage == "" {
			usage = fmt.Sprintf("set %s value", jsonTag)
		}

		// 根据字段类型注册不同的参数类型
		switch field.Kind() {
		case reflect.String:
			fs.StringVar(field.Addr().Interface().(*string), jsonTag, defaultValue, usage)
		case reflect.Int:
			defaultInt, err := strconv.Atoi(defaultValue)
			if err != nil {
				defaultInt = 0
			}
			fs.IntVar(field.Addr().Interface().(*int), jsonTag, defaultInt, usage)
		case reflect.Int64:
			defaultInt64, err := strconv.ParseInt(defaultValue, 10, 64)
			if err != nil {
				defaultInt64 = 0
			}
			fs.Int64Var(field.Addr().Interface().(*int64), jsonTag, defaultInt64, usage)
		case reflect.Bool:
			defaultBool := strings.ToLower(defaultValue) == "true"
			fs.BoolVar(field.Addr().Interface().(*bool), jsonTag, defaultBool, usage)
		case reflect.Float64:
			defaultFloat, err := strconv.ParseFloat(defaultValue, 64)
			if err != nil {
				defaultFloat = 0
			}
			fs.Float64Var(field.Addr().Interface().(*float64), jsonTag, defaultFloat, usage)
		default:
		}
	}
}
