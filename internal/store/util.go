package store

import "strings"

// isUniqueViolation 判断 SQLite 错误是否为唯一约束冲突。
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed")
}

// IsConstraintViolation 暴露给外部判断外键/约束冲突。
func IsConstraintViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "constraint failed") ||
		strings.Contains(msg, "FOREIGN KEY")
}
