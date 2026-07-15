package nsql

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

type LoadDataOrm struct {
	driverName          string
	filePath            string
	local               bool //是否是本地文件 默认false
	tableName           string
	character           string //字符集 默认utf8
	terminated          string //字段分隔符 默认,
	optionally_enclosed string //字段引用符 默认"
	lines_terminated    string //行分隔符 默认 \n
	ignore_lines        int    //忽略前几行 默认0
	columns             []string
}

func NewLoadDataOrm(filePath, tableName string) *LoadDataOrm {
	return &LoadDataOrm{filePath: filePath, tableName: tableName}
}

func (s *LoadDataOrm) DriverName(driverName string) *LoadDataOrm {
	s.driverName = driverName
	return s
}

func (s *LoadDataOrm) Local(local bool) *LoadDataOrm {
	s.local = local
	return s
}
func (s *LoadDataOrm) Character(character string) *LoadDataOrm {
	s.character = character
	return s
}
func (s *LoadDataOrm) Terminated(terminated string) *LoadDataOrm {
	s.terminated = terminated
	return s
}
func (s *LoadDataOrm) OptionallyEnclosed(optionally_enclosed string) *LoadDataOrm {
	s.optionally_enclosed = optionally_enclosed
	return s
}
func (s *LoadDataOrm) LinesTerminated(lines_terminated string) *LoadDataOrm {
	s.lines_terminated = lines_terminated
	return s
}
func (s *LoadDataOrm) IgnoreLines(ignore_lines int) *LoadDataOrm {
	s.ignore_lines = ignore_lines
	return s
}
func (s *LoadDataOrm) Columns(columns []string) *LoadDataOrm {
	s.columns = columns
	return s
}

func (s *LoadDataOrm) ToSql() (string, error) {
	if s.tableName == "" {
		return "", errors.New("tableName cannot be empty")
	}
	var sql bytes.Buffer
	local := ""
	if s.local {
		local = "LOCAL"
	}
	sql.WriteString(fmt.Sprintf("LOAD DATA %s INFILE '%s' \n", local, s.filePath))

	sql.WriteString(fmt.Sprintf("INTO TABLE %s\n", s.tableName))

	//FIELDS TERMINATED BY ','
	if s.terminated == "" {
		s.terminated = ","
	}
	sql.WriteString(fmt.Sprintf("FIELDS TERMINATED BY '%s'\n", s.terminated))

	//OPTIONALLY ENCLOSED BY '"'
	if s.optionally_enclosed == "" {
		s.optionally_enclosed = "\""
	}
	sql.WriteString(fmt.Sprintf("OPTIONALLY ENCLOSED BY '%s'\n", s.optionally_enclosed))

	//LINES TERMINATED BY '\n'
	if s.lines_terminated == "" {
		s.lines_terminated = "\\n"
	}
	sql.WriteString(fmt.Sprintf("LINES TERMINATED BY '%s'\n", s.lines_terminated))

	//IGNORE 1 LINES
	if s.ignore_lines > 0 {
		sql.WriteString(fmt.Sprintf("IGNORE %d LINES\n", s.ignore_lines))
	}

	// (column1, column2, column3)
	if len(s.columns) > 0 {

		sql.WriteString(fmt.Sprintf("(%s)\n", strings.Join(s.columns, ", ")))
	}
	return sql.String(), nil
}
