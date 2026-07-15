package nsql

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/go-lazyer/norm/driver"
)

type InsertOrm struct {
	driverName string
	insert     []map[string]any
	tableName  string
	primaryKey string
}

func NewInsertOrm() *InsertOrm {
	return &InsertOrm{driverName: driver.DRIVER_NAME_MYSQL}
}
func (s *InsertOrm) DriverName(driverName string) *InsertOrm {
	s.driverName = driverName
	return s
}
func (s *InsertOrm) Table(tableName string) *InsertOrm {
	s.tableName = tableName
	return s
}
func (s *InsertOrm) TableName() string {
	return s.tableName
}
func (s *InsertOrm) PrimaryKey(primaryKey string) *InsertOrm {
	s.primaryKey = primaryKey
	return s
}
func (s *InsertOrm) PrimaryKeyName() string {
	return s.primaryKey
}
func (s *InsertOrm) Insert(m ...map[string]any) *InsertOrm {
	s.insert = m
	return s
}
func (s *InsertOrm) ToSql(prepare bool) (string, []any, error) {

	if s.tableName == "" {
		return "", nil, errors.New("tableName cannot be empty")
	}
	if len(s.insert) == 0 {
		return "", nil, errors.New("insert cannot be empty")
	}
	params := make([]any, 0)
	phIdx := 1
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
		if s.primaryKey != "" {
			sql.WriteString(" RETURNING \"" + s.primaryKey + "\"")
		}
	}

	return sql.String(), params, nil
}
