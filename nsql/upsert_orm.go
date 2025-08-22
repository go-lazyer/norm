package nsql

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/go-lazyer/norm/driver"
	"github.com/go-lazyer/norm/nutil"
)

type UpsertOrm struct {
	insert    []map[string]any
	update    []string
	tableName string
}

func NewUpsertOrm() *UpsertOrm {
	return &UpsertOrm{}
}
func (s *UpsertOrm) Table(tableName string) *UpsertOrm {
	s.tableName = tableName
	return s
}
func (s *UpsertOrm) Insert(m ...map[string]any) *UpsertOrm {
	s.insert = m
	return s
}
func (s *UpsertOrm) Update(m []string) *UpsertOrm {
	s.update = m
	return s
}
func (s *UpsertOrm) ToSql(prepare bool) (string, []any, error) {

	if s.tableName == "" {
		return "", nil, errors.New("tableName cannot be empty")
	}
	if len(s.insert) == 0 {
		return "", nil, errors.New("insert cannot be empty")
	}
	if len(s.update) == 0 {
		return "", nil, errors.New("update cannot be empty")
	}
	params := make([]any, 0)
	var sql bytes.Buffer
	sql.WriteString("insert into " + s.tableName + " ")
	sql.WriteString("(")
	//把所有要修改的字段提取出来

	fields := nutil.Keys(s.insert[0])

	for n, field := range fields {
		if n != 0 {
			sql.WriteString(",")
		}
		sql.WriteString(" " + field + " ")
	}
	sql.WriteString(") values")

	for n, maps := range s.insert {
		if n != 0 {
			sql.WriteString(",")
		}
		sql.WriteString("(")
		for m, field := range fields {
			if m != 0 {
				sql.WriteString(",")
			}
			params = append(params, maps[field])
			if prepare {
				sql.WriteString(fmt.Sprintf(" %s ", driver.PLACE_HOLDER_GO))
			} else {
				sql.WriteString(fmt.Sprintf(" '%v' ", maps[field]))
			}
		}
		sql.WriteString(")")
	}
	sql.WriteString(" on duplicate key update ")
	for n, name := range s.update {
		if n != 0 {
			sql.WriteString(",")
		}
		sql.WriteString(fmt.Sprintf("%v=values(%v)", name, name))
	}
	return sql.String(), params, nil
}

func (s *UpsertOrm) ToPrepareSql() (string, [][]any, error) {

	if s.tableName == "" {
		return "", nil, errors.New("tableName cannot be empty")
	}
	if len(s.insert) == 0 {
		return "", nil, errors.New("insert cannot be empty")
	}
	if len(s.update) == 0 {
		return "", nil, errors.New("update cannot be empty")
	}

	var sql bytes.Buffer
	sql.WriteString("insert into " + s.tableName + " ")
	sql.WriteString("(")
	//把所有要修改的字段提取出来

	fields := nutil.Keys(s.insert[0])

	var valuesSql bytes.Buffer
	for n, field := range fields {
		if n != 0 {
			sql.WriteString(",")
			valuesSql.WriteString(",")
		}
		sql.WriteString(" " + field + " ")
		valuesSql.WriteString(" " + driver.PLACE_HOLDER_GO + " ")
	}
	sql.WriteString(") values (" + valuesSql.String() + ")")
	params := make([][]any, 0, len(s.insert))
	for _, maps := range s.insert {
		param := make([]any, 0)
		for _, field := range fields {
			param = append(param, maps[field])
		}
		params = append(params, param)
	}

	sql.WriteString(" on duplicate key update ")
	for n, name := range s.update {
		if n != 0 {
			sql.WriteString(",")
		}
		sql.WriteString(fmt.Sprintf("%v=values(%v)", name, name))
	}

	return sql.String(), params, nil
}
