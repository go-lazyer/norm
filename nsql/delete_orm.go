package nsql

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/go-lazyer/norm/constant"
)

type DeleteOrm struct {
	pageSize  int
	querys    []BaseQuery
	tableName string
}

func NewDeleteOrm() *DeleteOrm {
	return &DeleteOrm{}
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
	for i, query := range s.querys {
		if i != 0 {
			sql.WriteString(" or ")
		}
		source, param, _ := query.Source(s.tableName, prepare)
		sql.WriteString(" " + source + " ")
		params = append(params, param...)
	}
	if s.pageSize > 0 {
		sql.WriteString(fmt.Sprintf(" limit %s", constant.PLACE_HOLDER_GO))
		params = append(params, s.pageSize)
	}
	return sql.String(), params, nil
}
func (s *DeleteOrm) ToPrepareSql() (string, [][]any, error) {
	if s.tableName == "" {
		return "", nil, errors.New("tableName cannot be empty")
	}
	if len(s.querys) <= 0 {
		return "", nil, errors.New("the querys cannot be empty")
	}
	params := make([][]any, len(s.querys))
	var sql bytes.Buffer
	sql.WriteString("delete from " + s.tableName + " ")

	// 添加查询
	for n, query := range s.querys { //forr 会自动判断 querys 不为空
		if query == nil {
			continue
		}
		source, p, err := query.Source(s.tableName, true)
		if err != nil {
			return "", nil, err
		}
		if source == "" {
			continue
		}
		if n == 0 {
			sql.WriteString(" where   " + source + " ")
		}
		params[n] = append(params[n], p...)
	}

	if s.pageSize > 0 {
		sql.WriteString(fmt.Sprintf(" limit %s", constant.PLACE_HOLDER_GO))
		for i, param := range params {
			params[i] = append(param, s.pageSize)
		}
	}
	return sql.String(), params, nil
}
