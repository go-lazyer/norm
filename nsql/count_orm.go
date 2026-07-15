package nsql

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/go-lazyer/norm/driver"
)

type CountOrm struct {
	driverName string
	groupBy    []string //分组字段
	querys     []BaseQuery
	joins      []*Join
	columns    []string
	tableName  string
	tableAlias string
}

func NewCountOrm() *CountOrm {
	return &CountOrm{driverName: driver.DRIVER_NAME_MYSQL}
}
func (s *CountOrm) DriverName(driverName string) *CountOrm {
	s.driverName = driverName
	return s
}
func (s *CountOrm) Table(tableName string) *CountOrm {
	s.tableName = tableName
	s.tableAlias = tableName
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
	tableAlias := s.tableAlias
	if len(s.joins) == 0 { //单表查询，不需要表别名
		tableAlias = ""
	}

	params := make([]any, 0)
	phIdx := 1
	var sql bytes.Buffer
	sql.WriteString("select ")

	result := " count(*) count  "
	if len(s.columns) > 0 {
		result = s.columns[0]
	}

	sql.WriteString(result)
	sql.WriteString(" from  " + s.tableName + "")

	if tableAlias != "" {
		sql.WriteString(" " + tableAlias + " ")
	}

	for _, join := range s.joins {
		sql.WriteString(fmt.Sprintf(" %v %v on %v", join.joinType, join.tableName, join.condition))
		for i, query := range join.querys {
			if i == 0 {
				sql.WriteString(" and ")
			} else {
				sql.WriteString(" or ")
			}
			source, param, _ := query.Source(join.tableName, prepare, s.driverName, &phIdx)
			sql.WriteString(" " + source + " ")
			params = append(params, param...)
		}
	}

	for n, query := range s.querys {
		source, param, err := query.Source(tableAlias, prepare, s.driverName, &phIdx)
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
