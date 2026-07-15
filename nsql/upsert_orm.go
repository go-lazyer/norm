package nsql

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/go-lazyer/norm/driver"
)

type UpsertOrm struct {
	driverName string
	insert     []map[string]any
	update     []string
	tableName  string
	conflict   []string
}

func NewUpsertOrm() *UpsertOrm {
	return &UpsertOrm{driverName: driver.DRIVER_NAME_MYSQL}
}
func (s *UpsertOrm) DriverName(driverName string) *UpsertOrm {
	s.driverName = driverName
	return s
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
func (s *UpsertOrm) Conflict(cols ...string) *UpsertOrm {
	s.conflict = cols
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
	if driver.IsPostgres(s.driverName) && len(s.conflict) == 0 {
		return "", nil, errors.New("conflict columns cannot be empty for PostgreSQL")
	}
	params := make([]any, 0)
	var sql bytes.Buffer
	sql.WriteString("insert into " + s.tableName + " ")
	sql.WriteString("(")
	//把所有要修改的字段提取出来

	fields := MapKeys(s.insert[0])

	for n, field := range fields {
		if n != 0 {
			sql.WriteString(",")
		}
		sql.WriteString(" " + field + " ")
	}
	sql.WriteString(") values")

	phIdx := 1
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
				sql.WriteString(fmt.Sprintf(" %s ", nextPlaceholder(s.driverName, &phIdx)))
				
			} else {
				sql.WriteString(fmt.Sprintf(" '%v' ", maps[field]))
			}
		}
		sql.WriteString(")")
	}
	if driver.IsPostgres(s.driverName) {
		sql.WriteString(" on conflict (" + strings.Join(s.conflict, ",") + ") do update set ")
		for n, name := range s.update {
			if n != 0 {
				sql.WriteString(",")
			}
			sql.WriteString(fmt.Sprintf("%v=excluded.%v", name, name))
		}
	} else {
		sql.WriteString(" on duplicate key update ")
		for n, name := range s.update {
			if n != 0 {
				sql.WriteString(",")
			}
			sql.WriteString(fmt.Sprintf("%v=values(%v)", name, name))
		}
	}
	return sql.String(), params, nil
}