package nsql

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/go-lazyer/norm/driver"
)

type SelectOrm struct {
	orderBy    []string //排序字段
	groupBy    []string //分组字段
	pageStart  int
	pageSize   int
	pageNum    int
	querys     []BaseQuery
	joins      []*Join
	tableName  string
	tableAlias string
	columns    []string
}

func NewSelectOrm() *SelectOrm {
	return &SelectOrm{}
}
func (s *SelectOrm) Table(tableName string) *SelectOrm {
	s.tableName = tableName
	return s
}
func (s *SelectOrm) Where(query ...BaseQuery) *SelectOrm {
	if s.querys == nil {
		s.querys = make([]BaseQuery, 0)
	}
	s.querys = append(s.querys, query...)
	return s
}
func (s *SelectOrm) Join(join ...*Join) *SelectOrm {
	if s.joins == nil {
		s.joins = make([]*Join, 0)
	}
	s.joins = append(s.joins, join...)
	return s
}

// 表的别名
func (s *SelectOrm) TableAlias(tableName, tableAlias string) *SelectOrm {
	s.tableName = tableName
	s.tableAlias = tableAlias
	return s
}
func (s *SelectOrm) Result(columns ...string) *SelectOrm {
	s.columns = columns
	return s
}
func (s *SelectOrm) PageNum(pageNum int) *SelectOrm {
	s.pageNum = pageNum
	return s
}
func (s *SelectOrm) PageStart(pageStart int) *SelectOrm {
	s.pageStart = pageStart
	return s
}
func (s *SelectOrm) PageSize(pageSize int) *SelectOrm {
	s.pageSize = pageSize
	return s
}
func (s *SelectOrm) OrderBy(orderBy []string) *SelectOrm {
	s.orderBy = orderBy
	return s
}
func (s *SelectOrm) AddOrderBy(name string, orderByType string) *SelectOrm {
	if s.orderBy == nil {
		s.orderBy = make([]string, 0)
	}
	s.orderBy = append(s.orderBy, name+" "+orderByType)
	return s
}

func (s *SelectOrm) GroupBy(groupBy []string) *SelectOrm {
	s.groupBy = groupBy
	return s
}
func (s *SelectOrm) AddGroupBy(tableName, name string) *SelectOrm {
	if s.groupBy == nil {
		s.groupBy = make([]string, 0)
	}
	s.groupBy = append(s.groupBy, tableName+"."+name)
	return s
}

func (s *SelectOrm) ToSql(prepare bool) (string, []any, error) {
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
	if s.columns == nil {
		sql.WriteString(" * ")
	} else {
		sql.WriteString(strings.Join(s.columns, ","))
	}
	sql.WriteString(" from  " + s.tableName + "")

	if s.tableAlias != "" {
		sql.WriteString(" " + s.tableAlias + " ")
	}

	for _, join := range s.joins {
		table := join.tableName
		if join.tableAlias != "" {
			table = join.tableAlias
		}
		if join.tableName != "" {
			sql.WriteString(fmt.Sprintf(" %v %v %v on %v", join.joinType, join.tableName, join.tableAlias, join.condition))
		} else {
			sql.WriteString(fmt.Sprintf(" %v %v on %v", join.joinType, join.tableName, join.condition))
		}
		for i, query := range join.querys {
			if i == 0 {
				sql.WriteString(" and ")
			} else {
				sql.WriteString(" or ")
			}
			source, param, _ := query.Source(table, prepare)
			sql.WriteString(" " + source + " ")
			params = append(params, param...)
		}
	}

	for n, query := range s.querys {
		if query == nil {
			continue
		}
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

	for n, v := range s.groupBy {
		if n == 0 {
			sql.WriteString(" group by   ")
		} else {
			sql.WriteString(", ")
		}
		sql.WriteString(v)
	}
	for n, v := range s.orderBy {
		if n == 0 {
			sql.WriteString(" order by   ")
		} else {
			sql.WriteString(", ")

		}
		sql.WriteString(v)
	}
	if s.pageSize > 0 {
		if s.pageNum > 0 {
			s.pageStart = (s.pageNum - 1) * s.pageSize
		}
		params = append(params, s.pageSize, s.pageStart)
		if prepare {
			sql.WriteString(fmt.Sprintf(" limit %s offset %s", driver.PLACE_HOLDER_GO, driver.PLACE_HOLDER_GO))
		} else {
			sql.WriteString(fmt.Sprintf(" limit %d offset %d", s.pageSize, s.pageStart))
		}
	}

	return sql.String(), params, nil
}

func (s *SelectOrm) ToPrepareSql() (string, [][]any, error) {
	if s.tableName == "" {
		return "", nil, errors.New("tableName cannot be empty")
	}

	//设置表别名
	table := s.tableName
	if s.tableAlias != "" {
		table = s.tableAlias
	}

	var sql bytes.Buffer
	sql.WriteString("select ")
	if s.columns == nil {
		sql.WriteString(" * ")
	} else {
		sql.WriteString(strings.Join(s.columns, ","))
	}
	sql.WriteString(" from  " + s.tableName + "")

	if s.tableAlias != "" {
		sql.WriteString(" " + s.tableAlias + " ")
	}
	params := make([][]any, len(s.querys))
	for _, join := range s.joins { //forr 会自动判断joins不为空
		table := join.tableName
		if join.tableAlias != "" {
			table = join.tableAlias
		}
		if join.tableName != "" {
			sql.WriteString(fmt.Sprintf(" %v %v %v on %v", join.joinType, join.tableName, join.tableAlias, join.condition))
		} else {
			sql.WriteString(fmt.Sprintf(" %v %v on %v", join.joinType, join.tableName, join.condition))
		}
		for i, query := range join.querys {
			if i == 0 {
				sql.WriteString(" and ")
			} else {
				sql.WriteString(" or ")
			}
			source, p, _ := query.Source(table, true)
			sql.WriteString(" " + source + " ")

			for i, param := range params {
				params[i] = append(param, p...)
			}
		}
	}

	// 添加查询
	for n, query := range s.querys { //forr 会自动判断 querys 不为空
		if query == nil {
			continue
		}
		source, p, err := query.Source(table, true)
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

	// 添加分组
	for n, v := range s.groupBy {
		if n == 0 {
			sql.WriteString(" group by   ")
		} else {
			sql.WriteString(", ")
		}
		sql.WriteString(v)
	}

	// 添加排序
	for n, v := range s.orderBy {
		if n != 0 {
			sql.WriteString(" order by   ")
		} else {
			sql.WriteString(", ")

		}
		sql.WriteString(v)
	}
	if s.pageSize > 0 {
		if s.pageNum > 0 {
			s.pageStart = (s.pageNum - 1) * s.pageSize
		}
		sql.WriteString(fmt.Sprintf(" limit %s offset %s", driver.PLACE_HOLDER_GO, driver.PLACE_HOLDER_GO))
		for i, param := range params {
			params[i] = append(param, s.pageSize, s.pageStart)
		}
	}

	return sql.String(), params, nil
}
