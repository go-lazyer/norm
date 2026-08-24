package nsql

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/go-lazyer/norm/driver"
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

// GetFilePath returns the file path for the data file.
func (s *LoadDataOrm) GetFilePath() string { return s.filePath }

// GetTableName returns the target table name.
func (s *LoadDataOrm) GetTableName() string { return s.tableName }

// GetColumnList returns the column list for the load operation.
func (s *LoadDataOrm) GetColumnList() []string { return s.columns }

// GetTerminated returns the field terminator character.
func (s *LoadDataOrm) GetTerminated() string { return s.terminated }

// GetOptionallyEnclosed returns the field enclosure character.
func (s *LoadDataOrm) GetOptionallyEnclosed() string { return s.optionally_enclosed }

// GetLinesTerminated returns the line terminator string.
func (s *LoadDataOrm) GetLinesTerminated() string { return s.lines_terminated }

// GetIgnoreLines returns the number of lines to ignore at the start.
func (s *LoadDataOrm) GetIgnoreLines() int { return s.ignore_lines }

// GetLocal returns whether the file is a local (client-side) file.
func (s *LoadDataOrm) GetLocal() bool { return s.local }

func (s *LoadDataOrm) ToSql() (string, error) {
	if s.tableName == "" {
		return "", errors.New("tableName cannot be empty")
	}

	if driver.IsPostgres(s.driverName) && !s.local {
		return s.toPostgresSql()
	}
	return s.toMySql()
}

// toMySql 生成 MySQL LOAD DATA INFILE 语句。
func (s *LoadDataOrm) toMySql() (string, error) {
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

// toPostgresSql 生成 PostgreSQL COPY ... FROM STDIN 对应的服务端 COPY 语句。
func (s *LoadDataOrm) toPostgresSql() (string, error) {
	var sqlBuf strings.Builder
	sqlBuf.WriteString(fmt.Sprintf("COPY %s", s.tableName))
	if len(s.columns) > 0 {
		sqlBuf.WriteString(fmt.Sprintf(" (%s)", strings.Join(s.columns, ", ")))
	}
	sqlBuf.WriteString(fmt.Sprintf(" FROM '%s'", s.filePath))

	sqlBuf.WriteString(" WITH (FORMAT csv")
	if s.terminated != "" {
		sqlBuf.WriteString(fmt.Sprintf(", DELIMITER '%s'", s.terminated))
	}
	sqlBuf.WriteString(fmt.Sprintf(", HEADER %t", s.ignore_lines > 0))
	sqlBuf.WriteString(")")

	return sqlBuf.String(), nil
}
