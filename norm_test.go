package norm

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/go-lazyer/norm/driver"
	"github.com/go-lazyer/norm/nsql"
	// _ "github.com/go-sql-driver/mysql"
)

func GetDataSource() (DataSource, error) {
	username := "root"
	password := "******"
	host := "localhost"
	port := "3306"
	dbname := "north"
	connStr := fmt.Sprintf("%s:%s@(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", username, password, host, port, dbname)
	// 创建数据库连接
	return Open(driver.DRIVER_NAME_MYSQL, connStr, Config{MaxOpenConns: 10, MaxIdleConns: 10})
}

func TestCount(t *testing.T) {
	ds, err := GetDataSource()
	if err != nil {
		fmt.Print(err)
	}
	orm := nsql.NewCountOrm().Table("user")
	sqlStr, prarm, _ := orm.ToSql(true)
	fmt.Println(CountBySql(sqlStr, prarm, ds))
}

func TestPrepareCount(t *testing.T) {
	ds, err := GetDataSource()
	if err != nil {
		fmt.Print(err)
	}
	query := nsql.NewBetweenQuery("create_time", "2023-01-01 00:00:00", "2023-12-31 23:59:59")
	orm := nsql.NewCountOrm().Table("user").Where(query)
	sqlStr, prarms, _ := orm.ToPrepareSql()
	params := make([][]any, 0)
	params = append(params, []any{"2024-01-01 00:00:00", "2024-12-31 23:59:59"})
	params = append(params, []any{"2025-01-01 00:00:00", "2025-12-31 23:59:59"})
	res, _ := PrepareCountBySql(sqlStr, prarms, ds)
	fmt.Println(res)
}

type User struct {
	Id         sql.NullString `orm:"id" `             //
	Name       sql.NullString `orm:"name" default:""` // 昵称
	Age        sql.NullInt64  `orm:"age" default:""`  // 年龄
	CreateTime sql.NullTime   `orm:"create_time" `    // 用户名
}
type UserExtend struct {
	User
	Sex sql.NullString `orm:"sex" default:""` // 性别
}

func TestQueryByOrm(t *testing.T) {
	ds, err := GetDataSource()
	if err != nil {
		fmt.Print(err)
	}
	query := nsql.NewBetweenQuery("create_time", "2024-01-01 00:00:00", "2025-12-31 23:59:59")
	orm := nsql.NewSelectOrm().Table("user").Where(query).PageSize(10)
	users, err := QueryByOrm[UserExtend](orm, ds)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(users)
}

func TestInsert(t *testing.T) {
	ds, err := GetDataSource()
	if err != nil {
		fmt.Print(err)
	}
	//清空表
	_, err = DeleteByOrm(nsql.NewDeleteOrm().Table("user").Where(nsql.NewNotNullQuery("id")), ds)
	if err != nil {
		fmt.Print(err)
	}

	//插入单条数据
	m1 := map[string]any{
		"id":          1,
		"name":        "李一一",
		"sex":         "boy",
		"age":         "11",
		"create_time": "2021-01-01 00:00:00",
	}
	InsertByOrm(nsql.NewInsertOrm().Table("user").Insert(m1), ds)

	//插入多条数据
	m2 := map[string]any{
		"id":          2,
		"name":        "李二二",
		"sex":         "boy",
		"age":         "12",
		"create_time": "2022-01-01 00:00:00",
	}
	m3 := map[string]any{
		"id":          3,
		"name":        "李三三",
		"sex":         "boy",
		"age":         "13",
		"create_time": "2023-01-01 00:00:00",
	}
	InsertByOrm(nsql.NewInsertOrm().Table("user").Insert(m2, m3), ds)

	//批量插入数据
	m4 := map[string]any{
		"id":          4,
		"name":        "李四四",
		"sex":         "boy",
		"age":         "14",
		"create_time": "2024-01-01 00:00:00",
	}
	m5 := map[string]any{
		"id":          5,
		"name":        "李五五",
		"sex":         "boy",
		"age":         "15",
		"create_time": "2025-01-01 00:00:00",
	}
	m6 := map[string]any{
		"id":          6,
		"name":        "李六六",
		"sex":         "boy",
		"age":         "16",
		"create_time": "2026-01-01 00:00:00",
	}
	InsertsByOrm(nsql.NewInsertOrm().Table("user").Insert(m4, m5, m6), ds)
}

func TestUpdate(t *testing.T) {
	ds, err := GetDataSource()
	if err != nil {
		fmt.Print(err)
	}

	m1 := map[string]any{
		"sex": "girl",
		"age": "21",
	}
	UpdateByOrm(nsql.NewUpdateOrm().Table("user").Update(m1).Where(nsql.NewEqualQuery("name", "李一一")), ds)

	// UPDATE
	// 	USER
	// SET
	// 	sex = CASE
	// 	WHEN user.name = '李二二' THEN
	// 		girl
	// 	WHEN user.name = '李三三' THEN
	// 		girl
	// 	ELSE
	// 		sex
	// 	END,
	// 	age = CASE
	// 	WHEN user.name = '李二二' THEN
	// 		22
	// 	WHEN user.name = '李三三' THEN
	// 		23
	// 	ELSE
	// 		age
	// 	END
	// WHERE
	// 	user.name in('李二二', '李三三')
	m2 := map[string]any{
		"sex":        "girl",
		"age":        "22",
		"_condition": nsql.NewEqualQuery("name", "李二二"),
	}

	m3 := map[string]any{
		"sex":        "girl",
		"age":        "23",
		"_condition": nsql.NewEqualQuery("name", "李三三"),
	}
	UpdateByOrm(nsql.NewUpdateOrm().Table("user").Update(m2, m3).Where(nsql.NewInQuery("name", []string{"李二二", "李三三"})), ds)

	m4 := map[string]any{
		"sex": "girl",
		"age": "24",
	}

	m5 := map[string]any{
		"sex": "girl",
		"age": "25",
	}
	m6 := map[string]any{
		"sex": "girl",
		"age": "26",
	}

	q4 := nsql.NewEqualQuery("name", "李四四")
	q5 := nsql.NewEqualQuery("name", "李五五")
	q6 := nsql.NewEqualQuery("name", "李六六")

	UpdatesByOrm(nsql.NewUpdateOrm().Table("user").Update(m4, m5, m6).Where(q4, q5, q6), ds)

}

func TestUpsert(t *testing.T) {
	ds, err := GetDataSource()
	if err != nil {
		fmt.Print(err)
	}

	//插入单条数据
	m1 := map[string]any{
		"id":          1,
		"name":        "李11",
		"sex":         "boy",
		"age":         "21",
		"create_time": "2021-01-01 00:00:00",
	}

	UpsertByMap("user", m1, ds)
	//插入单条数据
	m7 := map[string]any{
		"id":          7,
		"name":        "李七七",
		"sex":         "boy",
		"age":         "17",
		"create_time": "2027-01-01 00:00:00",
	}

	UpsertByMap("user", m7, ds)

	//插入多条数据
	m2 := map[string]any{
		"id":          2,
		"name":        "李22",
		"sex":         "boy",
		"age":         "22",
		"create_time": "2022-01-01 00:00:00",
	}
	m8 := map[string]any{
		"id":          8,
		"name":        "李八八",
		"sex":         "boy",
		"age":         "18",
		"create_time": "2028-01-01 00:00:00",
	}
	UpsertsByMap("user", []map[string]any{m2, m8}, ds)
}

func TestDelete(t *testing.T) {
	ds, err := GetDataSource()
	if err != nil {
		fmt.Print(err)
	}
	query := nsql.NewBetweenQuery("create_time", "2023-01-01 00:00:00", "2025-12-31 23:59:59")
	orm := nsql.NewDeleteOrm().Table("user").Where(query).PageSize(2)
	sqlStr, param, _ := orm.ToSql(true)
	fmt.Print(sqlStr, param)
	fmt.Println(DeleteBySql(sqlStr, param, ds))
}

// func TestPrepareDelete(t *testing.T) {
// 	ds, err := GetDataSource()
// 	if err != nil {
// 		fmt.Print(err)
// 	}
// 	query1 := nsql.NewBetweenQuery("create_time", "2023-01-01 00:00:00", "2023-12-31 23:59:59")
// 	query2 := nsql.NewBetweenQuery("create_time", "2024-01-01 00:00:00", "2024-12-31 23:59:59")
// 	orm := nsql.NewDeleteOrm().Table("user").Where(query1, query2).PageSize(1)
// 	sqlStr, prarms, _ := orm.ToPrepareSql()

// 	fmt.Print(sqlStr, prarms)

//		res, err := PrepareDelete(sqlStr, prarms, ds)
//		if err != nil {
//			fmt.Println(err)
//		}
//		fmt.Println(res)
//	}
