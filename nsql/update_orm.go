package nsql

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/go-lazyer/norm/constant"
	"github.com/go-lazyer/north/nmap"
)

type UpdateOrm struct {
	querys     []BaseQuery
	update     []map[string]any
	joins      []*Join
	tableName  string
	tableAlias string
}

func NewUpdateOrm() *UpdateOrm {
	return &UpdateOrm{}
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
	var sql bytes.Buffer
	sql.WriteString("update " + s.tableName + " set ")
	n := 0
	if len(s.update) > 1 { //批量更新
		//把所有要修改的字段提取出来
		fields := make(map[string]string)
		for _, setMap := range s.update {
			for name := range setMap {
				if name == constant.CONDITION {
					continue
				}
				fields[name] = ""
			}
		}

		for field := range fields {
			if n != 0 {
				sql.WriteString(",")
			}
			sql.WriteString(fmt.Sprintf("%v = CASE ", field))
			for _, setMap := range s.update {
				condition, ok := setMap[constant.CONDITION]
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

				source, param, err := baseQuery.Source(s.tableName, prepare)
				if err != nil {
					return "", nil, err
				}
				params = append(params, param...)

				if prepare {
					sql.WriteString(fmt.Sprintf(" WHEN %v THEN %v", source, constant.PLACE_HOLDER_GO))
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
				sql.WriteString(fmt.Sprintf("%v=%s", name, constant.PLACE_HOLDER_GO))
			} else {
				sql.WriteString(fmt.Sprintf("%v='%v'", name, value))
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

// 用于批量更新，单独更新不建议使用
// 生成updatePrepareSql 是orm 中的 update 和 querys 中len的关系为，一对多，多对一，或者n对n 只有这3种情况
// 多个update 的长度和key必须一致
// 多个querys 的长度和query必须一致

func (s *UpdateOrm) ToPrepareSql() (string, [][]any, error) {

	if s.tableName == "" {
		return "", nil, errors.New("tableName  cannot be empty")
	}

	if len(s.update) == 0 {
		return "", nil, errors.New("the update cannot be empty")
	}
	if len(s.querys) == 0 {
		return "", nil, errors.New("the querys cannot be empty")
	}

	if len(s.update) > 1 && len(s.querys) > 1 && len(s.update) != len(s.querys) {
		return "", nil, errors.New("the update and querys must be one to one or many to one")
	}

	table := s.tableName
	if s.tableAlias != "" {
		table = s.tableAlias
	}

	params := make([][]any, max(len(s.update), len(s.querys)))

	var sql bytes.Buffer
	sql.WriteString("update " + s.tableName + " set ")

	fields := nmap.Keys(s.update[0])

	for i, update := range s.update {
		for n, field := range fields {
			value, ok := update[field]
			if !ok {
				return "", nil, fmt.Errorf("missing value for field %s in update map. ", field)
			}

			if i == 0 { //多个update ，只把第一个拼写到sql 中
				if n != 0 {
					sql.WriteString(",")
				}
				sql.WriteString(fmt.Sprintf("%v=%s", field, constant.PLACE_HOLDER_GO))
			}

			if len(s.update) == 1 {
				for i, param := range params {
					params[i] = append(param, value)
				}
			} else {
				params[i] = append(params[i], value)
			}
		}
	}
	// 添加查询
	source0, _, _ := s.querys[0].Source(table, true)
	for n, query := range s.querys { //forr 会自动判断 querys 不为空
		if query == nil {
			continue
		}
		source, p, err := query.Source(table, true)
		if err != nil {
			return "", nil, err
		}
		//querys中结构必须完全相同
		if source == "" || source != source0 {
			return "", nil, errors.New("the structure of query must be exactly the same")
		}
		if n == 0 {
			sql.WriteString(" where   " + source + " ")
		}
		if len(s.querys) == 1 {
			for i, param := range params {
				params[i] = append(param, p...)
			}
		} else {
			params[n] = append(params[n], p...)
		}
	}

	return sql.String(), params, nil
}
