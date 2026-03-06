package nsql

import (
	"bytes"
	"fmt"

	"github.com/go-lazyer/norm/driver"
)

type BaseQuery interface {
	Source(table string, prepare bool) (string, []any, error)
}

type NullQuery struct {
	table string
	field string
}

func NewNullQuery(field string) *NullQuery {
	return &NullQuery{field: field}
}
func NewNullQueryWithTable(table, field string) *NullQuery {
	return &NullQuery{table: table, field: field}
}

func (q *NullQuery) Source(table string, prepare bool) (string, []any, error) {
	fieldStr := "`" + q.field + "`"
	if table != "" {
		fieldStr = table + "." + fieldStr
	}
	return fieldStr + " is null", nil, nil
}

type NotNullQuery struct {
	table string
	field string
}

func NewNotNullQuery(field string) *NotNullQuery {
	return &NotNullQuery{field: field}
}
func NewNotNullQueryWithTable(table, field string) *NotNullQuery {
	return &NotNullQuery{table: table, field: field}
}

func (q *NotNullQuery) Source(table string, prepare bool) (string, []any, error) {
	fieldStr := "`" + q.field + "`"
	if table != "" {
		fieldStr = table + "." + fieldStr
	}
	return fieldStr + " is not null", nil, nil
}

type BetweenQuery struct {
	field       string
	firstValue  any
	secondValue any
}

func NewBetweenQuery(field string, firstValue any, secondValue any) *BetweenQuery {
	return &BetweenQuery{field: field, firstValue: firstValue, secondValue: secondValue}
}

func (q *BetweenQuery) Source(table string, prepare bool) (string, []any, error) {
	param := []any{q.firstValue, q.secondValue}

	fieldStr := "`" + q.field + "`"
	if table != "" {
		fieldStr = table + "." + fieldStr
	}
	if prepare {
		return fmt.Sprintf("%s between %s and %s", fieldStr, driver.PLACE_HOLDER_GO, driver.PLACE_HOLDER_GO), param, nil
	}
	if isNumeric(q.firstValue) {
		return fmt.Sprintf("%s between %v and %v", fieldStr, q.firstValue, q.secondValue), param, nil
	} else {
		return fmt.Sprintf("%s between '%v' and '%v' ", fieldStr, q.firstValue, q.secondValue), param, nil
	}
}

type NotBetweenQuery struct {
	field       string
	firstValue  any
	secondValue any
}

func NewNotBetweenQuery(field string, firstValue any, secondValue any) *NotBetweenQuery {
	return &NotBetweenQuery{field: field, firstValue: firstValue, secondValue: secondValue}
}

func (q *NotBetweenQuery) Source(table string, prepare bool) (string, []any, error) {
	param := []any{q.firstValue, q.secondValue}
	fieldStr := "`" + q.field + "`"
	if table != "" {
		fieldStr = table + "." + fieldStr
	}
	if prepare {
		return fmt.Sprintf("%s not between %s and %s", fieldStr, driver.PLACE_HOLDER_GO, driver.PLACE_HOLDER_GO), param, nil
	}
	if isNumeric(q.firstValue) {
		return fmt.Sprintf("%s not between %v and %v", fieldStr, q.firstValue, q.secondValue), param, nil
	} else {
		return fmt.Sprintf("%s not between '%v' and '%v' ", fieldStr, q.firstValue, q.secondValue), param, nil
	}
}

type EqualQuery struct {
	table string
	field string
	value any
}

func NewEqualQuery(field string, value any) *EqualQuery {
	return &EqualQuery{field: field, value: value}
}
func NewEqualQueryWithTable(table, field string, value any) *EqualQuery {
	return &EqualQuery{table: table, field: field, value: value}
}

func (q *EqualQuery) Source(table string, prepare bool) (string, []any, error) {
	fieldStr := "`" + q.field + "`"
	if table != "" {
		fieldStr = table + "." + fieldStr
	}
	if prepare {
		return fmt.Sprintf("%s = %s", fieldStr, driver.PLACE_HOLDER_GO), []any{q.value}, nil
	}
	if isNumeric(q.value) {
		return fmt.Sprintf("%s = %v", fieldStr, q.value), []any{q.value}, nil
	} else {
		return fmt.Sprintf("%s = '%v'", fieldStr, q.value), []any{q.value}, nil
	}
}

type NotEqualQuery struct {
	field string
	value any
}

func NewNotEqualQuery(field string, value any) *NotEqualQuery {
	return &NotEqualQuery{field: field, value: value}
}

func (q *NotEqualQuery) Source(table string, prepare bool) (string, []any, error) {
	fieldStr := "`" + q.field + "`"
	if table != "" {
		fieldStr = table + "." + fieldStr
	}
	if prepare {
		return fmt.Sprintf("%s != %s", fieldStr, driver.PLACE_HOLDER_GO), []any{q.value}, nil
	}
	if isNumeric(q.value) {
		return fmt.Sprintf("%s != %v", fieldStr, q.value), []any{q.value}, nil
	} else {
		return fmt.Sprintf("%s != '%v'", fieldStr, q.value), []any{q.value}, nil
	}
}

type InQuery[T comparable] struct {
	field string
	value []T // 使用泛型类型参数 T 代替 any
}

// 泛型构造函数
func NewInQuery[T comparable](field string, values []T) *InQuery[T] {
	return &InQuery[T]{
		field: field,
		value: values,
	}
}
func (q *InQuery[T]) Source(table string, prepare bool) (string, []any, error) {
	fieldStr := "`" + q.field + "`"
	if table != "" {
		fieldStr = table + "." + fieldStr
	}
	var sql bytes.Buffer
	sql.WriteString(fieldStr + " in (")
	for k, v := range q.value {
		if k != 0 {
			sql.WriteString(" ,")
		}
		if prepare {
			sql.WriteString(fmt.Sprintf(" %s", driver.PLACE_HOLDER_GO))
		} else if isNumeric(v) {
			sql.WriteString(fmt.Sprintf(" %v ", v))
		} else {
			sql.WriteString(fmt.Sprintf(" '%v' ", v))
		}

	}
	sql.WriteString(")")
	// return sql.String(), q.value, nil
	// Convert []T to []any for the return value
	params := make([]any, len(q.value))
	for i, v := range q.value {
		params[i] = v
	}
	return sql.String(), params, nil
}

type NotInQuery[T comparable] struct {
	field string
	value []T
}

func NewNotInQuery[T comparable](field string, value []T) *NotInQuery[T] {
	return &NotInQuery[T]{field: field, value: value}
}

func (q *NotInQuery[T]) Source(table string, prepare bool) (string, []any, error) {
	fieldStr := "`" + q.field + "`"
	if table != "" {
		fieldStr = table + "." + fieldStr
	}

	var sql bytes.Buffer
	sql.WriteString(fieldStr + " not in (")
	for k, v := range q.value {
		if k != 0 {
			sql.WriteString(" ,")
		}
		if prepare {
			sql.WriteString(fmt.Sprintf(" %s", driver.PLACE_HOLDER_GO))
		} else if isNumeric(q.value) {
			sql.WriteString(fmt.Sprintf(" %v ", v))
		} else {
			sql.WriteString(fmt.Sprintf(" '%v' ", v))
		}

	}
	sql.WriteString(")")
	// return sql.String(), q.value, nil
	params := make([]any, len(q.value))
	for i, v := range q.value {
		params[i] = v
	}
	return sql.String(), params, nil
}

type LikeQuery struct {
	field string
	value any
}

func NewLikeQuery(field string, value any) *LikeQuery {
	return &LikeQuery{field: field, value: value}
}

func (q *LikeQuery) Source(table string, prepare bool) (string, []any, error) {
	fieldStr := "`" + q.field + "`"
	if table != "" {
		fieldStr = table + "." + fieldStr
	}
	if prepare {
		return fmt.Sprintf("%s like %s", fieldStr, driver.PLACE_HOLDER_GO), []any{q.value}, nil
	}
	if isNumeric(q.value) {
		return fmt.Sprintf("%s like '%v'", fieldStr, q.value), []any{q.value}, nil
	} else {
		return fmt.Sprintf("%s like '%v'", fieldStr, q.value), []any{q.value}, nil
	}
}

type NotLikeQuery struct {
	field string
	value any
}

func NewNotLikeQuery(field string, value any) *NotLikeQuery {
	return &NotLikeQuery{field: field, value: value}
}

func (q *NotLikeQuery) Source(table string, prepare bool) (string, []any, error) {
	fieldStr := "`" + q.field + "`"
	if table != "" {
		fieldStr = table + "." + fieldStr
	}
	if prepare {
		return fmt.Sprintf("%s not like %s", fieldStr, driver.PLACE_HOLDER_GO), []any{q.value}, nil
	}
	if isNumeric(q.value) {
		return fmt.Sprintf("%s not like %v", fieldStr, q.value), []any{q.value}, nil
	} else {
		return fmt.Sprintf("%s not like '%v'", fieldStr, q.value), []any{q.value}, nil
	}
}

type GreaterThanQuery struct {
	field string
	value any
}

func NewGreaterThanQuery(field string, value any) *GreaterThanQuery {
	return &GreaterThanQuery{field: field, value: value}
}

func (q *GreaterThanQuery) Source(table string, prepare bool) (string, []any, error) {
	fieldStr := "`" + q.field + "`"
	if table != "" {
		fieldStr = table + "." + fieldStr
	}
	if prepare {
		return fmt.Sprintf("%s > %s", fieldStr, driver.PLACE_HOLDER_GO), []any{q.value}, nil
	}
	if isNumeric(q.value) {
		return fmt.Sprintf("%s > %v", fieldStr, q.value), []any{q.value}, nil
	} else {
		return fmt.Sprintf("%s > '%v'", fieldStr, q.value), []any{q.value}, nil
	}
}

type GreaterThanOrEqualQuery struct {
	field string
	value any
}

func NewGreaterThanOrEqualQuery(field string, value any) *GreaterThanOrEqualQuery {
	return &GreaterThanOrEqualQuery{field: field, value: value}
}

func (q *GreaterThanOrEqualQuery) Source(table string, prepare bool) (string, []any, error) {
	fieldStr := "`" + q.field + "`"
	if table != "" {
		fieldStr = table + "." + fieldStr
	}
	if prepare {
		return fmt.Sprintf("%s >= %s", fieldStr, driver.PLACE_HOLDER_GO), []any{q.value}, nil
	}
	if isNumeric(q.value) {
		return fmt.Sprintf("%s >= %v", fieldStr, q.value), []any{q.value}, nil
	} else {
		return fmt.Sprintf("%s >= '%v'", fieldStr, q.value), []any{q.value}, nil
	}
}

type LessThanQuery struct {
	field string
	value any
}

func NewLessThanQuery(field string, value any) *LessThanQuery {
	return &LessThanQuery{field: field, value: value}
}

func (q *LessThanQuery) Source(table string, prepare bool) (string, []any, error) {
	fieldStr := "`" + q.field + "`"
	if table != "" {
		fieldStr = table + "." + fieldStr
	}
	if prepare {
		return fmt.Sprintf("%s < %s", fieldStr, driver.PLACE_HOLDER_GO), []any{q.value}, nil
	}
	if isNumeric(q.value) {
		return fmt.Sprintf("%s < %v", fieldStr, q.value), []any{q.value}, nil
	} else {
		return fmt.Sprintf("%s < '%v'", fieldStr, q.value), []any{q.value}, nil
	}
}

type LessThanOrEqualQuery struct {
	field string
	value any
}

func NewLessThanOrEqualQuery(field string, value any) *LessThanOrEqualQuery {
	return &LessThanOrEqualQuery{field: field, value: value}
}

func (q *LessThanOrEqualQuery) Source(table string, prepare bool) (string, []any, error) {
	fieldStr := "`" + q.field + "`"
	if table != "" {
		fieldStr = table + "." + fieldStr
	}
	if prepare {
		return fmt.Sprintf("%s <= %s", fieldStr, driver.PLACE_HOLDER_GO), []any{q.value}, nil
	}
	if isNumeric(q.value) {
		return fmt.Sprintf("%s <= %v", fieldStr, q.value), []any{q.value}, nil
	} else {
		return fmt.Sprintf("%s <= '%v'", fieldStr, q.value), []any{q.value}, nil
	}
}

type FieldEqualQuery struct {
	firstTable  string
	firstField  string
	secondTable string
	secondField string
}

func NewFieldEqualQuery(firstTable, firstField, secondTable, secondField string) *FieldEqualQuery {
	return &FieldEqualQuery{firstTable: firstTable, firstField: firstField, secondTable: secondTable, secondField: secondField}
}

func (q *FieldEqualQuery) Source(table string, prepare bool) (string, []any, error) {
	firstFieldStr := "`" + q.firstField + "`"
	if q.firstTable != "" {
		firstFieldStr = q.firstTable + "." + firstFieldStr
	} else if table != "" {
		firstFieldStr = table + "." + firstFieldStr
	}

	secondFieldStr := "`" + q.secondField + "`"
	if q.secondTable != "" {
		secondFieldStr = q.secondTable + "." + secondFieldStr
	} else if table != "" {
		secondFieldStr = table + "." + secondFieldStr
	}
	return fmt.Sprintf("%s = %s", firstFieldStr, secondFieldStr), []any{}, nil
}

type FieldNotEqualQuery struct {
	firstTable  string
	firstField  string
	secondTable string
	secondField string
}

func NewFieldNotEqualQuery(firstTable, firstField, secondTable, secondField string) *FieldNotEqualQuery {
	return &FieldNotEqualQuery{firstTable: firstTable, firstField: firstField, secondTable: secondTable, secondField: secondField}
}

func (q *FieldNotEqualQuery) Source(table string, prepare bool) (string, []any, error) {
	firstFieldStr := "`" + q.firstField + "`"
	if q.firstTable != "" {
		firstFieldStr = q.firstTable + "." + firstFieldStr
	} else if table != "" {
		firstFieldStr = table + "." + firstFieldStr
	}

	secondFieldStr := "`" + q.secondField + "`"
	if q.secondTable != "" {
		secondFieldStr = q.secondTable + "." + secondFieldStr
	} else if table != "" {
		secondFieldStr = table + "." + secondFieldStr
	}
	return fmt.Sprintf("%s != %s", firstFieldStr, secondFieldStr), []any{}, nil
}

type FieldGreaterThanQuery struct {
	firstTable  string
	firstField  string
	secondTable string
	secondField string
}

func NewFieldGreaterThanQuery(firstTable, firstField, secondTable, secondField string) *FieldGreaterThanQuery {
	return &FieldGreaterThanQuery{firstTable: firstTable, firstField: firstField, secondTable: secondTable, secondField: secondField}
}

func (q *FieldGreaterThanQuery) Source(table string, prepare bool) (string, []any, error) {
	firstFieldStr := "`" + q.firstField + "`"
	if q.firstTable != "" {
		firstFieldStr = q.firstTable + "." + firstFieldStr
	} else if table != "" {
		firstFieldStr = table + "." + firstFieldStr
	}

	secondFieldStr := "`" + q.secondField + "`"
	if q.secondTable != "" {
		secondFieldStr = q.secondTable + "." + secondFieldStr
	} else if table != "" {
		secondFieldStr = table + "." + secondFieldStr
	}

	return fmt.Sprintf("%s > %s", firstFieldStr, secondFieldStr), []any{}, nil
}

type FieldGreaterThanOrEqualQuery struct {
	firstTable  string
	firstField  string
	secondTable string
	secondField string
}

func NewFieldGreaterThanOrEqualQuery(firstTable, firstField, secondTable, secondField string) *FieldGreaterThanOrEqualQuery {
	return &FieldGreaterThanOrEqualQuery{firstTable: firstTable, firstField: firstField, secondTable: secondTable, secondField: secondField}
}

func (q *FieldGreaterThanOrEqualQuery) Source(table string, prepare bool) (string, []any, error) {
	firstFieldStr := "`" + q.firstField + "`"
	if q.firstTable != "" {
		firstFieldStr = q.firstTable + "." + firstFieldStr
	} else if table != "" {
		firstFieldStr = table + "." + firstFieldStr
	}

	secondFieldStr := "`" + q.secondField + "`"
	if q.secondTable != "" {
		secondFieldStr = q.secondTable + "." + secondFieldStr
	} else if table != "" {
		secondFieldStr = table + "." + secondFieldStr
	}

	return fmt.Sprintf("%s >= %s", firstFieldStr, secondFieldStr), []any{}, nil
}

type FieldLessThanQuery struct {
	firstTable  string
	firstField  string
	secondTable string
	secondField string
}

func NewFieldLessThanQuery(firstTable, firstField, secondTable, secondField string) *FieldLessThanQuery {
	return &FieldLessThanQuery{firstTable: firstTable, firstField: firstField, secondTable: secondTable, secondField: secondField}
}

func (q *FieldLessThanQuery) Source(table string, prepare bool) (string, []any, error) {
	firstFieldStr := "`" + q.firstField + "`"
	if q.firstTable != "" {
		firstFieldStr = q.firstTable + "." + firstFieldStr
	} else if table != "" {
		firstFieldStr = table + "." + firstFieldStr
	}

	secondFieldStr := "`" + q.secondField + "`"
	if q.secondTable != "" {
		secondFieldStr = q.secondTable + "." + secondFieldStr
	} else if table != "" {
		secondFieldStr = table + "." + secondFieldStr
	}

	return fmt.Sprintf("%s < %s", firstFieldStr, secondFieldStr), []any{}, nil
}

type FieldLessThanOrEqualQuery struct {
	firstTable  string
	firstField  string
	secondTable string
	secondField string
}

func NewFieldLessThanOrEqualQuery(firstTable, firstField, secondTable, secondField string) *FieldLessThanOrEqualQuery {
	return &FieldLessThanOrEqualQuery{firstTable: firstTable, firstField: firstField, secondTable: secondTable, secondField: secondField}
}

func (q *FieldLessThanOrEqualQuery) Source(table string, prepare bool) (string, []any, error) {
	firstFieldStr := "`" + q.firstField + "`"
	if q.firstTable != "" {
		firstFieldStr = q.firstTable + "." + firstFieldStr
	} else if table != "" {
		firstFieldStr = table + "." + firstFieldStr
	}

	secondFieldStr := "`" + q.secondField + "`"
	if q.secondTable != "" {
		secondFieldStr = q.secondTable + "." + secondFieldStr
	} else if table != "" {
		secondFieldStr = table + "." + secondFieldStr
	}
	return fmt.Sprintf("%s <= %s", firstFieldStr, secondFieldStr), []any{}, nil
}

// field end
type BoolQuery struct {
	query []BaseQuery
}

func NewBoolQuery() *BoolQuery {
	return &BoolQuery{
		query: make([]BaseQuery, 0),
	}
}

func (q *BoolQuery) And(queries ...BaseQuery) *BoolQuery {
	q.query = append(q.query, queries...)
	return q
}

func (q *BoolQuery) Source(table string, prepare bool) (string, []any, error) {
	params := make([]any, 0)
	if len(q.query) <= 0 {
		return "", params, nil
	}
	var sql bytes.Buffer
	sql.WriteString("(")
	if len(q.query) > 0 {
		for k, query := range q.query {
			if k != 0 {
				sql.WriteString(" and")
			}
			source, param, err := query.Source(table, prepare)
			if err != nil {
				return "", params, err
			}
			params = append(params, param...)
			sql.WriteString(" " + source + " ")
		}
	}
	sql.WriteString(")")
	return sql.String(), params, nil
}
