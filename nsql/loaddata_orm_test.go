package nsql_test

import (
	"fmt"
	"testing"

	"github.com/go-lazyer/norm/nsql"
)

func TestLoadDataOrm_ToSql(t *testing.T) {
	s := nsql.NewLoadDataOrm("/data/1.txt", "s_tech_info")
	s.Columns([]string{"id", "name", "age"})
	fmt.Println(s.ToSql())
}
