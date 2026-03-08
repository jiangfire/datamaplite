package utils

import "time"

// Ptr 返回值的指针
func Ptr[T any](v T) *T {
	return &v
}

// Contains 检查切片中是否包含指定值
func Contains[T comparable](slice []T, value T) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// Filter 过滤切片
func Filter[T any](slice []T, predicate func(T) bool) []T {
	result := make([]T, 0, len(slice))
	for _, v := range slice {
		if predicate(v) {
			result = append(result, v)
		}
	}
	return result
}

// Map 映射切片
func Map[T any, R any](slice []T, mapper func(T) R) []R {
	result := make([]R, len(slice))
	for i, v := range slice {
		result[i] = mapper(v)
	}
	return result
}

// TimeToPtr 将time.Time转换为*time.Time（如果不是零值）
func TimeToPtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// DefaultString 如果字符串为空则返回默认值
func DefaultString(s, defaultValue string) string {
	if s == "" {
		return defaultValue
	}
	return s
}

// DefaultInt 如果值为0则返回默认值
func DefaultInt(v, defaultValue int) int {
	if v == 0 {
		return defaultValue
	}
	return v
}

// StringSliceToMap 将字符串切片转换为map（用于快速查找）
func StringSliceToMap(slice []string) map[string]bool {
	m := make(map[string]bool, len(slice))
	for _, s := range slice {
		m[s] = true
	}
	return m
}
