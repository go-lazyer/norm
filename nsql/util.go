package nsql

import (
	"fmt"

	"github.com/go-lazyer/norm/driver"
)

// nextPlaceholder returns the parameter placeholder for the given driver at the current index and increments it.
// PostgreSQL uses $1, $2, ...; MySQL/Manticore use ?.
func nextPlaceholder(driverName string, phIdx *int) string {
	p := "?"
	if driver.IsPostgres(driverName) {
		p = fmt.Sprintf("$%d", *phIdx)
	}
	*phIdx++
	return p
}

// quoteField wraps a field name with the appropriate identifier quote for the driver.
// PostgreSQL uses double quotes ("field"); MySQL/Manticore use backticks (`field`).
func quoteField(driverName, field string) string {
	if driver.IsPostgres(driverName) {
		return `"` + field + `"`
	}
	return "`" + field + "`"
}

// isNumeric 检查一个值是否为数字类型
func isNumeric(inter any) bool {
	if inter == nil {
		return false
	}
	switch inter.(type) {
	case int:
		return true
	case int8:
		return true
	case int16:
		return true
	case int32:
		return true
	case int64:
		return true
	case float32:
		return true
	case float64:
		return true
	default:
		return false
	}
}
func MapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
