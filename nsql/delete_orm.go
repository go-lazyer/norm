package nsql

import (
	"bytes"
	"errors"
	"fmt"
)

type DeleteOrm struct {
	driverName string
	pageSize   int
	querys     []BaseQuery
	tableName  string
}

func NewDeleteOrm() *DeleteOrm {
	return &DeleteOrm{}
}
func (s *DeleteOrm) DriverName(driverName string) *DeleteOrm {
	s.driverName = driverName
	return s
}
func (s *DeleteOrm) Table(tableName string) *DeleteOrm {
	s.tableName = tableName
	return s
}
func (s *DeleteOrm) Where(query ...BaseQuery) *DeleteOrm {
	if s.querys == nil {
		s.querys = make([]BaseQuery, 0)
	}
	s.querys = append(s.querys, query...)
	return s
}
func (s *DeleteOrm) PageSize(pageSize int) *DeleteOrm {
	s.pageSize = pageSize
	return s
}

func (s *DeleteOrm) ToSql(prepare bool) (string, []any, error) {
	if s.tableName == "" {
		return "", nil, errors.New("tableName cannot be empty")
	}
	if s.querys == nil || len(s.querys) != 1 {
		return "", nil, errors.New("the querys size must be 1")
	}
	params := make([]any, 0)
	var sql bytes.Buffer
	sql.WriteString("delete from " + s.tableName + " ")

	sql.WriteString(" where   ")
	phIdx := 1
	for i, query := range s.querys {
		if i != 0 {
			sql.WriteString(" or ")
		}
		source, param, _ := query.Source(s.tableName, prepare, s.driverName, &phIdx)
		sql.WriteString(" " + source + " ")
		params = append(params, param...)
	}
	if s.pageSize > 0 {
		sql.WriteString(fmt.Sprintf(" limit %s", nextPlaceholder(s.driverName, &phIdx)))
		params = append(params, s.pageSize)
	}
	return sql.String(), params, nil
}
