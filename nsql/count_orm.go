package nsql

import (
	"bytes"
	"errors"
	"fmt"
)

type CountOrm struct {
	groupBy    []string //分组字段
	querys     []BaseQuery
	joins      []*Join
	columns    []string
	tableName  string
	tableAlias string
}

func NewCountOrm() *CountOrm {
	return &CountOrm{}
}
func (s *CountOrm) Table(tableName string) *CountOrm {
	s.tableName = tableName
	return s
}
func (s *CountOrm) Where(query ...BaseQuery) *CountOrm {
	if s.querys == nil {
		s.querys = make([]BaseQuery, 0)
	}
	s.querys = append(s.querys, query...)
	return s
}
func (s *CountOrm) Join(join ...*Join) *CountOrm {
	if s.joins == nil {
		s.joins = make([]*Join, 0)
	}
	s.joins = append(s.joins, join...)
	return s
}

// 表的别名
func (s *CountOrm) TableAlias(tableName, tableAlias string) *CountOrm {
	s.tableName = tableName
	s.tableAlias = tableAlias
	return s
}
func (s *CountOrm) Result(columns ...string) *CountOrm {
	s.columns = columns
	return s
}

func (s *CountOrm) GroupBy(groupBy []string) *CountOrm {
	s.groupBy = groupBy
	return s
}
func (s *CountOrm) AddGroupBy(tableName, name string) *CountOrm {
	if s.groupBy == nil {
		s.groupBy = make([]string, 0)
	}
	s.groupBy = append(s.groupBy, tableName+"."+name)
	return s
}

func (s *CountOrm) ToSql(prepare bool) (string, []any, error) {
	if s.tableName == "" {
		return "", nil, errors.New("tableName cannot be empty")
	}
	table := s.tableName
	if s.tableAlias != "" {
		table = s.tableAlias
	}

	params := make([]any, 0)
	var sql bytes.Buffer
	sql.WriteString("select ")

	result := " count(*) count  "
	if len(s.columns) > 0 {
		result = s.columns[0]
	}

	sql.WriteString(result)
	sql.WriteString(" from  " + s.tableName + "")

	if s.tableAlias != "" {
		sql.WriteString(" " + s.tableAlias + " ")
	}

	for _, join := range s.joins {
		sql.WriteString(fmt.Sprintf(" %v %v on %v", join.joinType, join.tableName, join.condition))
		for i, query := range join.querys {
			if i == 0 {
				sql.WriteString(" and ")
			} else {
				sql.WriteString(" or ")
			}
			source, param, _ := query.Source(join.tableName, prepare)
			sql.WriteString(" " + source + " ")
			params = append(params, param...)
		}
	}

	for n, query := range s.querys {
		source, param, err := query.Source(table, prepare)
		if err != nil {
			return "", nil, err
		}
		if source == "" {
			continue
		}
		if n == 0 {
			sql.WriteString(" where   ")
		} else {
			sql.WriteString(" or ")
		}
		sql.WriteString(" " + source + " ")
		params = append(params, param...)
	}

	return sql.String(), params, nil
}

func (s *CountOrm) ToPrepareSql() (string, [][]any, error) {
	if s.tableName == "" {
		return "", nil, errors.New("tableName cannot be empty")
	}

	table := s.tableName
	if s.tableAlias != "" {
		table = s.tableAlias
	}

	var sql bytes.Buffer
	sql.WriteString("select ")

	result := " count(*) count  "
	if len(s.columns) > 0 {
		result = s.columns[0]
	}

	sql.WriteString(result)
	sql.WriteString(" from  " + s.tableName + "")

	if s.tableAlias != "" {
		sql.WriteString(" " + s.tableAlias + " ")
	}
	params := make([][]any, len(s.querys))
	for _, join := range s.joins {
		sql.WriteString(fmt.Sprintf(" %v %v on %v", join.joinType, join.tableName, join.condition))
		for i, query := range join.querys {
			if i == 0 {
				sql.WriteString(" and ")
			} else {
				sql.WriteString(" or ")
			}
			source, p, _ := query.Source(join.tableName, true)
			sql.WriteString(" " + source + " ")
			for i, param := range params {
				params[i] = append(param, p...)
			}
		}
	}

	for n, query := range s.querys {
		source, p, err := query.Source(table, true)
		if err != nil {
			return "", nil, err
		}
		if source == "" {
			continue
		}
		if n == 0 {
			sql.WriteString(" where   ")
		} else {
			sql.WriteString(" or ")
		}
		sql.WriteString(" " + source + " ")
		params[n] = append(params[n], p...)
	}

	return sql.String(), params, nil
}
