package nsql

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/go-lazyer/norm/driver"
)

const (
	CONDITION = "_condition" //
)

type UpdateOrm struct {
	driverName string
	querys     []BaseQuery
	update     []map[string]any
	joins      []*Join
	tableName  string
	tableAlias string
}

func NewUpdateOrm() *UpdateOrm {
	return &UpdateOrm{driverName: driver.DRIVER_NAME_MYSQL}
}
func (s *UpdateOrm) DriverName(driverName string) *UpdateOrm {
	s.driverName = driverName
	return s
}
func (s *UpdateOrm) Table(tableName string) *UpdateOrm {
	s.tableName = tableName
	return s
}
func (s *UpdateOrm) Where(query ...BaseQuery) *UpdateOrm {
	if s.querys == nil {
		s.querys = make([]BaseQuery, 0)
	}
	s.querys = append(s.querys, query...)
	return s
}
func (s *UpdateOrm) Update(m ...map[string]any) *UpdateOrm {
	s.update = m
	return s
}
func (s *UpdateOrm) Join(join ...*Join) *UpdateOrm {
	if s.joins == nil {
		s.joins = make([]*Join, 0)
	}
	s.joins = append(s.joins, join...)
	return s
}

// 表的别名
func (s *UpdateOrm) TableAlias(tableName, tableAlias string) *UpdateOrm {
	s.tableName = tableName
	s.tableAlias = tableAlias
	return s
}

func (s *UpdateOrm) ToSql(prepare bool) (string, []any, error) {

	if s.tableName == "" {
		return "", nil, errors.New("tableName  cannot be empty")
	}

	if len(s.querys) == 0 {
		return "", nil, errors.New("the querys cannot be empty")
	}

	table := s.tableName
	if s.tableAlias != "" {
		table = s.tableAlias
	}

	params := make([]any, 0)
	phIdx := 1
	var sql bytes.Buffer
	sql.WriteString("update " + s.tableName + " set ")
	n := 0
	if len(s.update) > 1 { //批量更新
		//把所有要修改的字段提取出来
		fields := make(map[string]string)
		for _, setMap := range s.update {
			for name := range setMap {
				if name == CONDITION {
					continue
				}
				fields[name] = ""
			}
		}

		for field := range fields {
			if n != 0 {
				sql.WriteString(",")
			}
			sql.WriteString(fmt.Sprintf("%s = CASE ", quoteField(s.driverName, field)))
			for _, setMap := range s.update {
				condition, ok := setMap[CONDITION]
				if !ok {
					return "", nil, errors.New("missing condition in update map for batch update")
				}

				baseQuery, ok := condition.(BaseQuery)
				if !ok {
					return "", nil, errors.New("condition in update map is not of type query")
				}

				v, ok := setMap[field]
				if !ok {
					continue
				}

				source, param, err := baseQuery.Source(s.tableName, prepare, s.driverName, &phIdx)
				if err != nil {
					return "", nil, err
				}
				params = append(params, param...)

				if prepare {
					sql.WriteString(fmt.Sprintf(" WHEN %v THEN %v", source, nextPlaceholder(s.driverName, &phIdx)))
				} else {
					sql.WriteString(fmt.Sprintf(" WHEN %v THEN %v", source, v))
				}
				params = append(params, v)
			}
			sql.WriteString(" else " + field)
			sql.WriteString(" END ")
			n++
		}
	} else { //单个更新
		for name, value := range s.update[0] {
			if n != 0 {
				sql.WriteString(",")
			}
			if prepare {
				sql.WriteString(fmt.Sprintf("%s=%s", quoteField(s.driverName, name), nextPlaceholder(s.driverName, &phIdx)))
			} else {
				sql.WriteString(fmt.Sprintf("%s='%v'", quoteField(s.driverName, name), value))
			}
			params = append(params, value)
			n++
		}
	}

	// 添加查询
	for n, query := range s.querys {
		if query == nil {
			continue
		}
		source, param, err := query.Source(table, prepare, s.driverName, &phIdx)
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