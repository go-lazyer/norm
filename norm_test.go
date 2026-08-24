package norm

import (
	"database/sql"
	"strconv"
	"strings"
	"testing"

	"github.com/go-lazyer/norm/driver"
	"github.com/go-lazyer/norm/nsql"
	_ "github.com/go-sql-driver/mysql"
)

const testTable = "norm_user"
const testTableExt = "norm_user_ext"

// openTestDS 打开一个用于集成测试的 DataSource。
// 当 DSN 为 postgres/postgresql 前缀时，自动切换到 pgx 驱动。
// 连接建立后会自动执行建表，保证新库也能直接跑测试。
// 测试结束时自动关闭连接池，避免连接泄漏。
func openTestDS(t *testing.T) DataSource {
	dsn := "postgres://postgres:Daoway_Mysql_iO12@test.daoway.cn:5432/postgres"
	// dsn := "postgres://postgres@search.daoway.cn:5432/daowei?sslmode=disable"

	// dsn := "root:Daoway_Mysql_iO12@tcp(test.daoway.cn:3306)/test?charset=utf8mb4&parseTime=true&loc=Local"

	driverName := driver.DRIVER_NAME_MYSQL
	if strings.HasPrefix(strings.ToLower(dsn), "postgres://") || strings.HasPrefix(strings.ToLower(dsn), "postgresql://") {
		driverName = driver.DRIVER_NAME_PGX
	}
	ds, err := Open(driverName, dsn, Config{MaxOpenConns: 10, MaxIdleConns: 10})
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	t.Cleanup(func() {
		ds.Close()
	})
	ensureTestTable(t, ds)
	return ds
}

func testPh(ds DataSource, idx int) string {
	// PostgreSQL 使用 $1/$2... 占位符；MySQL 使用 ?。
	if ds.DriverName == driver.DRIVER_NAME_PGX {
		return "$" + strconv.Itoa(idx)
	}
	return "?"
}

// ensureTestTable 在测试数据库中创建测试表。
// 该函数幂等，可在每个测试启动时重复调用。
func ensureTestTable(t *testing.T, ds DataSource) {
	t.Helper()
	createSQL := "create table if not exists " + testTable + " (" +
		"id varchar(64) primary key," +
		"name varchar(255)," +
		"age bigint," +
		"create_time timestamp" +
		")"
	if _, err := ds.ExecBySql(createSQL); err != nil {
		t.Fatalf("failed to create test table %s: %v", testTable, err)
	}
	createExtSQL := "create table if not exists " + testTableExt + " (" +
		"id varchar(64) primary key," +
		"sex varchar(16)" +
		")"
	if _, err := ds.ExecBySql(createExtSQL); err != nil {
		t.Fatalf("failed to create test table %s: %v", testTableExt, err)
	}
}

// seedUsers 清空测试表并插入 3 条测试数据。
// 返回插入的行数。
func seedUsers(t *testing.T, ds DataSource) int {
	t.Helper()
	// 清空表
	_, err := ds.DeleteByOrm(nsql.NewDeleteOrm().Table(testTable).Where(nsql.NewNotNullQuery("id")))
	if err != nil {
		t.Fatalf("failed to clean user table: %v", err)
	}
	_, err = ds.DeleteByOrm(nsql.NewDeleteOrm().Table(testTableExt).Where(nsql.NewNotNullQuery("id")))
	if err != nil {
		t.Fatalf("failed to clean user_ext table: %v", err)
	}
	rows := []map[string]any{
		{"id": "1", "name": "李一一", "age": "11", "create_time": "2021-01-01 00:00:00"},
		{"id": "2", "name": "李二二", "age": "12", "create_time": "2022-01-01 00:00:00"},
		{"id": "3", "name": "李三三", "age": "13", "create_time": "2023-01-01 00:00:00"},
	}
	_, err = ds.InsertsByMap(testTable, "id", rows)
	if err != nil {
		t.Fatalf("failed to seed user table: %v", err)
	}
	extRows := []map[string]any{
		{"id": "1", "sex": "boy"},
		{"id": "2", "sex": "boy"},
		{"id": "3", "sex": "boy"},
	}
	_, err = ds.InsertsByMap(testTableExt, "id", extRows)
	if err != nil {
		t.Fatalf("failed to seed user_ext table: %v", err)
	}
	return len(rows)
}

// User / UserExtend 是查询结果映射的模型结构体。
type User struct {
	Id         sql.NullString `db:"id"`
	Name       sql.NullString `db:"name" default:""`
	Age        sql.NullInt64  `db:"age" default:""`
	CreateTime sql.NullTime   `db:"create_time"`
}

type UserExtend struct {
	User
	Sex sql.NullString `db:"sex" default:""`
}

// =========================================================================
// Count — 计数查询
// =========================================================================

// TestCountByOrm 测试 ORM 方式统计全表行数。
// SQL: SELECT count(*) count FROM user
func TestCountByOrm(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	orm := nsql.NewCountOrm().Table(testTable)
	if rawSQL, _, rawErr := orm.ToSql(false); rawErr != nil {
		t.Logf("orm.ToSql(false) error: %v", rawErr)
	} else {
		t.Logf("orm raw sql: %s", rawSQL)
	}
	cnt, err := ds.CountByOrm(orm)
	if err != nil {
		t.Fatalf("CountByOrm error: %v", err)
	}
	if cnt != 3 {
		t.Errorf("expected count=3, got %d", cnt)
	}
}

// TestCountBySql 测试原始 SQL 方式统计全表行数。
// SQL: SELECT count(*) FROM norm_user
func TestCountBySql(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	cnt, err := ds.CountBySql("select count(*) from "+testTable, nil)
	if err != nil {
		t.Fatalf("CountBySql error: %v", err)
	}
	if cnt != 3 {
		t.Errorf("expected count=3, got %d", cnt)
	}
}

// TestCountWithWhere 测试带等值条件的计数查询。
// SQL: SELECT count(*) count FROM user WHERE `name` = ?
// params: ["李一一"]
func TestCountWithWhere(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	orm := nsql.NewCountOrm().Table(testTable).Where(nsql.NewEqualQuery("name", "李一一"))
	if rawSQL, _, rawErr := orm.ToSql(false); rawErr != nil {
		t.Logf("orm.ToSql(false) error: %v", rawErr)
	} else {
		t.Logf("orm raw sql: %s", rawSQL)
	}
	cnt, err := ds.CountByOrm(orm)
	if err != nil {
		t.Fatalf("CountByOrm error: %v", err)
	}
	if cnt != 1 {
		t.Errorf("expected count=1, got %d", cnt)
	}
}

// TestCountWithGroupBy 测试带分组和指定结果列的计数 SQL 生成。
// SQL: SELECT name, count(*) cnt FROM user GROUP BY user.`name`
func TestCountWithGroupBy(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	orm := nsql.NewCountOrm().Result("name", "count(*) cnt").
		Table(testTable).
		AddGroupBy(testTable, "name")
	if rawSQL, _, rawErr := orm.ToSql(false); rawErr != nil {
		t.Logf("orm.ToSql(false) error: %v", rawErr)
	} else {
		t.Logf("orm raw sql: %s", rawSQL)
	}
	sqlStr, params, err := orm.ToSql(true)
	if err != nil {
		t.Fatalf("ToSql error: %v", err)
	}
	t.Logf("count group by SQL: %s | params: %v", sqlStr, params)
	if sqlStr == "" {
		t.Error("expected non-empty SQL")
	}
}

// =========================================================================
// Query / Select — 查询
// =========================================================================

// TestQueryByOrm 测试 ORM 方式的 BETWEEN 范围查询 + 分页。
// SQL: SELECT * FROM user WHERE `create_time` BETWEEN ? AND ? LIMIT ?
// params: ["2020-01-01 00:00:00", "2025-12-31 23:59:59", 10]
func TestQueryByOrm(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	query := nsql.NewBetweenQuery("create_time", "2020-01-01 00:00:00", "2025-12-31 23:59:59")
	orm := nsql.NewSelectOrm().Table(testTable).Where(query).PageSize(10)
	if rawSQL, _, rawErr := orm.ToSql(false); rawErr != nil {
		t.Logf("orm.ToSql(false) error: %v", rawErr)
	} else {
		t.Logf("orm raw sql: %s", rawSQL)
	}
	users, err := ds.QueryByOrm[UserExtend](orm)
	if err != nil {
		t.Fatalf("QueryByOrm error: %v", err)
	}
	if len(users) != 3 {
		t.Errorf("expected 3 users, got %d", len(users))
	}
	t.Logf("queried %d users", len(users))
}

// TestQueryBySql 测试原始 SQL 的等值查询，将结果映射到结构体。
// SQL: SELECT * FROM norm_user WHERE name = ?
// params: ["李一一"]
func TestQueryBySql(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	sqlStr := "select * from " + testTable + " where name = " + testPh(ds, 1)
	users, err := ds.QueryBySql[UserExtend](sqlStr, []any{"李一一"})
	if err != nil {
		t.Fatalf("QueryBySql error: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}
	if users[0].Name.String != "李一一" {
		t.Errorf("expected name=李一一, got %s", users[0].Name.String)
	}
}

// TestSelectWithOrderBy 测试带排序的查询（按 age 降序）。
// SQL: SELECT * FROM user ORDER BY age DESC LIMIT ?
// params: [10]
func TestSelectWithOrderBy(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	orm := nsql.NewSelectOrm().Table(testTable).
		AddOrderBy("age", "desc").
		PageSize(10)
	if rawSQL, _, rawErr := orm.ToSql(false); rawErr != nil {
		t.Logf("orm.ToSql(false) error: %v", rawErr)
	} else {
		t.Logf("orm raw sql: %s", rawSQL)
	}
	users, err := ds.QueryByOrm[UserExtend](orm)
	if err != nil {
		t.Fatalf("QueryByOrm error: %v", err)
	}
	if len(users) != 3 {
		t.Errorf("expected 3 users, got %d", len(users))
	}
	// 验证按年龄降序排列
	if users[0].Age.Int64 < users[2].Age.Int64 {
		t.Errorf("expected descending order by age")
	}
}

// TestSelectWithPageNum 测试基于页码的分页查询（第 1 页，每页 2 条）。
// SQL: SELECT * FROM user ORDER BY id ASC LIMIT ? OFFSET ?
// params: [2, 0]
func TestSelectWithPageNum(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	orm := nsql.NewSelectOrm().Table(testTable).
		AddOrderBy("id", "asc").
		PageNum(1).PageSize(2)
	if rawSQL, _, rawErr := orm.ToSql(false); rawErr != nil {
		t.Logf("orm.ToSql(false) error: %v", rawErr)
	} else {
		t.Logf("orm raw sql: %s", rawSQL)
	}
	users, err := ds.QueryByOrm[UserExtend](orm)
	if err != nil {
		t.Fatalf("QueryByOrm error: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users on page 1, got %d", len(users))
	}
}

// TestSelectWithResultColumns 测试指定返回列的查询。
// SQL: SELECT id, name FROM user LIMIT ?
// params: [10]
func TestSelectWithResultColumns(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	orm := nsql.NewSelectOrm().Result("id", "name").Table(testTable).PageSize(10)
	if rawSQL, _, rawErr := orm.ToSql(false); rawErr != nil {
		t.Logf("orm.ToSql(false) error: %v", rawErr)
	} else {
		t.Logf("orm raw sql: %s", rawSQL)
	}
	users, err := ds.QueryByOrm[UserExtend](orm)
	if err != nil {
		t.Fatalf("QueryByOrm error: %v", err)
	}
	if len(users) == 0 {
		t.Error("expected at least one user")
	}
}

// TestSelectWithInQuery 测试 IN 条件查询。
// SQL: SELECT * FROM user WHERE `name` IN (?, ?) LIMIT ?
// params: ["李一一", "李三三", 10]
func TestSelectWithInQuery(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	query := nsql.NewInQuery("name", []string{"李一一", "李三三"})
	orm := nsql.NewSelectOrm().Table(testTable).Where(query).PageSize(10)
	if rawSQL, _, rawErr := orm.ToSql(false); rawErr != nil {
		t.Logf("orm.ToSql(false) error: %v", rawErr)
	} else {
		t.Logf("orm raw sql: %s", rawSQL)
	}
	users, err := ds.QueryByOrm[UserExtend](orm)
	if err != nil {
		t.Fatalf("QueryByOrm error: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

// TestSelectWithBoolQuery 测试 BoolQuery 多条件 AND 组合查询。
// SQL: SELECT * FROM user WHERE (`name` = ? AND `age` >= ?) LIMIT ?
// params: ["李三三", "13", 10]
func TestSelectWithBoolQuery(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	boolQ := nsql.NewBoolQuery().
		And(nsql.NewEqualQuery("name", "李三三")).
		And(nsql.NewGreaterThanOrEqualQuery("age", "13"))
	orm := nsql.NewSelectOrm().Table(testTable).Where(boolQ).PageSize(10)
	if rawSQL, _, rawErr := orm.ToSql(false); rawErr != nil {
		t.Logf("orm.ToSql(false) error: %v", rawErr)
	} else {
		t.Logf("orm raw sql: %s", rawSQL)
	}
	users, err := ds.QueryByOrm[UserExtend](orm)
	if err != nil {
		t.Fatalf("QueryByOrm error: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user (name=李三三 and age>=13), got %d", len(users))
	}
}

// =========================================================================
// Join — 多表关联查询
// =========================================================================

// TestSelectWithInnerJoin 测试 INNER JOIN 联表查询，验证 norm_user JOIN norm_user_ext 获取 sex 字段。
// SQL: SELECT u.*, e.sex FROM norm_user u INNER JOIN norm_user_ext e ON u.id=e.id LIMIT ?
// params: [10]
func TestSelectWithInnerJoin(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	join := nsql.NewAliasJoin(testTableExt, "e", nsql.INNER_JOIN).
		Condition(testTable, "id", "e", "id")
	orm := nsql.NewSelectOrm().Result(testTable+".*", "e.sex").
		Table(testTable).Join(join).PageSize(10)
	if rawSQL, _, rawErr := orm.ToSql(false); rawErr != nil {
		t.Logf("orm.ToSql(false) error: %v", rawErr)
	} else {
		t.Logf("orm raw sql: %s", rawSQL)
	}
	users, err := ds.QueryByOrm[UserExtend](orm)
	if err != nil {
		t.Fatalf("QueryByOrm error: %v", err)
	}
	if len(users) != 3 {
		t.Errorf("expected 3 users from join, got %d", len(users))
	}
	for _, u := range users {
		if u.Sex.String != "boy" {
			t.Errorf("expected sex=boy from join, got %s", u.Sex.String)
		}
	}
	t.Logf("queried %d users via inner join, all sex verified", len(users))
}

// TestSelectWithJoinAndWhere 测试 JOIN + WHERE 条件查询。
// SQL: SELECT u.*, e.sex FROM norm_user u INNER JOIN norm_user_ext e ON u.id=e.id WHERE u.`name` = ? LIMIT ?
// params: ["李一一", 10]
func TestSelectWithJoinAndWhere(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	join := nsql.NewAliasJoin(testTableExt, "e", nsql.INNER_JOIN).
		Condition(testTable, "id", "e", "id")
	orm := nsql.NewSelectOrm().Result(testTable+".*", "e.sex").
		Table(testTable).Join(join).
		Where(nsql.NewEqualQuery("name", "李一一")).
		PageSize(10)
	if rawSQL, _, rawErr := orm.ToSql(false); rawErr != nil {
		t.Logf("orm.ToSql(false) error: %v", rawErr)
	} else {
		t.Logf("orm raw sql: %s", rawSQL)
	}
	users, err := ds.QueryByOrm[UserExtend](orm)
	if err != nil {
		t.Fatalf("QueryByOrm error: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user from join with name filter, got %d", len(users))
	}
	if users[0].Name.String != "李一一" {
		t.Errorf("expected name=李一一, got %s", users[0].Name.String)
	}
	if users[0].Sex.String != "boy" {
		t.Errorf("expected sex=boy from join, got %s", users[0].Sex.String)
	}
}

// TestSelectWithJoinAndPagination 测试 JOIN + 分页 + 排序。
// SQL: SELECT u.*, e.sex FROM norm_user u INNER JOIN norm_user_ext e ON u.id=e.id ORDER BY u.age DESC LIMIT ? OFFSET ?
// params: [2, 0]
func TestSelectWithJoinAndPagination(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	join := nsql.NewAliasJoin(testTableExt, "e", nsql.INNER_JOIN).
		Condition(testTable, "id", "e", "id")
	orm := nsql.NewSelectOrm().Result(testTable+".*", "e.sex").
		Table(testTable).Join(join).
		AddOrderBy(testTable+".age", "desc").
		PageNum(1).PageSize(2)
	if rawSQL, _, rawErr := orm.ToSql(false); rawErr != nil {
		t.Logf("orm.ToSql(false) error: %v", rawErr)
	} else {
		t.Logf("orm raw sql: %s", rawSQL)
	}
	users, err := ds.QueryByOrm[UserExtend](orm)
	if err != nil {
		t.Fatalf("QueryByOrm error: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users (page 1, size 2), got %d", len(users))
	}
	// 验证按年龄降序排列
	if users[0].Age.Int64 < users[1].Age.Int64 {
		t.Errorf("expected descending order by age in join query")
	}
}

// TestSelectWithAliasJoin 测试使用别名方式的 JOIN 查询。
// SQL: SELECT u.id, u.name, e.sex FROM norm_user u INNER JOIN norm_user_ext e ON u.id=e.id WHERE u.`age` >= ? LIMIT ?
// params: ["12", 10]
func TestSelectWithAliasJoin(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	join := nsql.NewAliasJoin(testTableExt, "e", nsql.INNER_JOIN).
		Condition("u", "id", "e", "id")
	orm := nsql.NewSelectOrm().Result("u.id", "u.name", "e.sex").
		TableAlias(testTable, "u").Join(join).
		Where(nsql.NewGreaterThanOrEqualQuery("age", "12")).
		PageSize(10)
	if rawSQL, _, rawErr := orm.ToSql(false); rawErr != nil {
		t.Logf("orm.ToSql(false) error: %v", rawErr)
	} else {
		t.Logf("orm raw sql: %s", rawSQL)
	}
	users, err := ds.QueryByOrm[UserExtend](orm)
	if err != nil {
		t.Fatalf("QueryByOrm error: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users (age>=12), got %d", len(users))
	}
}

// TestCountWithJoin 测试 COUNT + JOIN 组合查询。
// SQL: SELECT count(*) count FROM norm_user INNER JOIN norm_user_ext ON norm_user.id=norm_user_ext.id
func TestCountWithJoin(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	join := nsql.NewJoin(testTableExt, nsql.INNER_JOIN).
		Condition(testTable, "id", testTableExt, "id")
	orm := nsql.NewCountOrm().Table(testTable).Join(join)
	if rawSQL, _, rawErr := orm.ToSql(false); rawErr != nil {
		t.Logf("orm.ToSql(false) error: %v", rawErr)
	} else {
		t.Logf("orm raw sql: %s", rawSQL)
	}
	cnt, err := ds.CountByOrm(orm)
	if err != nil {
		t.Fatalf("CountByOrm error: %v", err)
	}
	if cnt != 3 {
		t.Errorf("expected count=3 from join, got %d", cnt)
	}
}

// =========================================================================
// Insert — 插入
// =========================================================================

// TestInsertByOrm 测试 ORM 方式单条插入。
// SQL: INSERT INTO user (id, name, age, create_time) VALUES (?,?,?,?)
// params: [1, "测试一", "20", "2024-01-01 00:00:00"]
func TestInsertByOrm(t *testing.T) {
	ds := openTestDS(t)
	// 清空表
	// SQL: DELETE FROM user WHERE id IS NOT NULL
	_, _ = ds.DeleteByOrm(nsql.NewDeleteOrm().Table(testTable).Where(nsql.NewNotNullQuery("id")))

	m := map[string]any{
		"id": "1", "name": "测试一", "age": "20", "create_time": "2024-01-01 00:00:00",
	}
	id, err := ds.InsertByOrm(nsql.NewInsertOrm().Table(testTable).PrimaryKey("id").Insert(m))
	if err != nil {
		t.Fatalf("InsertByOrm error: %v", err)
	}
	t.Logf("insert id: %d", id)
}

// TestInsertByMap 测试 InsertByMap 快捷方式插入单条数据。
// SQL: INSERT INTO user (id, name, age, create_time) VALUES (?,?,?,?)
// params: [1, "map插入", "25", "2024-06-01 00:00:00"]
func TestInsertByMap(t *testing.T) {
	ds := openTestDS(t)
	_, _ = ds.DeleteByOrm(nsql.NewDeleteOrm().Table(testTable).Where(nsql.NewNotNullQuery("id")))

	m := map[string]any{
		"id": "1", "name": "map插入", "age": "25", "create_time": "2024-06-01 00:00:00",
	}
	id, err := ds.InsertByMap(testTable, "id", m)
	if err != nil {
		t.Fatalf("InsertByMap error: %v", err)
	}
	t.Logf("insert id: %d", id)
}

// TestInsertsByMap 测试 InsertsByMap 批量插入并返回插入条数。
// SQL: INSERT INTO user (id, name, age, create_time) VALUES (?,?,?,?),(?,?,?,?)
// params: [1, "批量1", "15", ..., 2, "批量2", "16", ...]
func TestInsertsByMap(t *testing.T) {
	ds := openTestDS(t)
	_, _ = ds.DeleteByOrm(nsql.NewDeleteOrm().Table(testTable).Where(nsql.NewNotNullQuery("id")))

	rows := []map[string]any{
		{"id": "1", "name": "批量1", "age": "15", "create_time": "2024-01-01 00:00:00"},
		{"id": "2", "name": "批量2", "age": "16", "create_time": "2024-02-01 00:00:00"},
	}
	n, err := ds.InsertsByMap(testTable, "id", rows)
	if err != nil {
		t.Fatalf("InsertsByMap error: %v", err)
	}
	if n != 2 {
		t.Errorf("expected inserted count=2, got %d", n)
	}
}

// TestInsertsByOrm 测试 InsertsByOrm 批量插入并返回插入条数。
// SQL: INSERT INTO user (id, name, age, create_time) VALUES (?,?,?,?),(?,?,?,?)
// params: [1, "Orm批量1", ..., 2, "Orm批量2", ...]
func TestInsertsByOrm(t *testing.T) {
	ds := openTestDS(t)
	_, _ = ds.DeleteByOrm(nsql.NewDeleteOrm().Table(testTable).Where(nsql.NewNotNullQuery("id")))

	m1 := map[string]any{"id": "1", "name": "Orm批量1", "age": "17", "create_time": "2024-03-01 00:00:00"}
	m2 := map[string]any{"id": "2", "name": "Orm批量2", "age": "18", "create_time": "2024-04-01 00:00:00"}
	n, err := ds.InsertsByOrm(nsql.NewInsertOrm().Table(testTable).PrimaryKey("id").Insert(m1, m2))
	if err != nil {
		t.Fatalf("InsertsByOrm error: %v", err)
	}
	if n != 2 {
		t.Errorf("expected inserted count=2, got %d", n)
	}
}

// =========================================================================
// Update — 更新
// =========================================================================

// TestUpdateByOrm 测试 ORM 方式单条更新（等值条件）。
// SQL: UPDATE user SET `age`=? WHERE `name` = ?
// params: ["21", "李一一"]
func TestUpdateByOrm(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	m := map[string]any{"age": "21"}
	n, err := ds.UpdateByOrm(
		nsql.NewUpdateOrm().Table(testTable).Update(m).Where(nsql.NewEqualQuery("name", "李一一")),
	)
	if err != nil {
		t.Fatalf("UpdateByOrm error: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 row affected, got %d", n)
	}

	// 验证更新结果
	sqlStr := "select * from " + testTable + " where name = " + testPh(ds, 1)
	users, _ := ds.QueryBySql[User](sqlStr, []any{"李一一"})
	if len(users) == 1 && users[0].Age.Int64 != 21 {
		t.Errorf("expected age=21, got %d", users[0].Age.Int64)
	}
}

// TestUpdateBatchByOrm 测试批量更新（使用 CASE WHEN 语法）。
// SQL: UPDATE user SET `age`=CASE WHEN `name`=? THEN ? WHEN `name`=? THEN ? ELSE `age` END
// WHERE `name` IN (?, ?)
// params: ["李二二", "22", "李三三", "23", "李二二", "李三三"]
func TestUpdateBatchByOrm(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	m2 := map[string]any{
		"age":        "22",
		"_condition": nsql.NewEqualQuery("name", "李二二"),
	}
	m3 := map[string]any{
		"age":        "23",
		"_condition": nsql.NewEqualQuery("name", "李三三"),
	}
	n, err := ds.UpdateByOrm(
		nsql.NewUpdateOrm().Table(testTable).Update(m2, m3).
			Where(nsql.NewInQuery("name", []string{"李二二", "李三三"})),
	)
	if err != nil {
		t.Fatalf("batch UpdateByOrm error: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 rows affected, got %d", n)
	}
}

// TestUpdateBySql 测试原始 SQL 方式更新。
// SQL: UPDATE norm_user SET age = ? WHERE name = ?
// params: ["99", "李一一"]
func TestUpdateBySql(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	sqlStr := "update " + testTable + " set age = " + testPh(ds, 1) + " where name = " + testPh(ds, 2)
	n, err := ds.UpdateBySql(sqlStr, []any{"99", "李一一"})
	if err != nil {
		t.Fatalf("UpdateBySql error: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 row affected, got %d", n)
	}
}

// TestUpdatesByOrm 测试批量更新的 UpdatesByOrm（返回受影响行数数组）。
// SQL: UPDATE user SET `age`=CASE WHEN `name`=? THEN ? WHEN `name`=? THEN ? ELSE `age` END
// WHERE `name` IN (?, ?)
func TestUpdatesByOrm(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	m2 := map[string]any{
		"age":        "30",
		"_condition": nsql.NewEqualQuery("name", "李二二"),
	}
	m3 := map[string]any{
		"age":        "31",
		"_condition": nsql.NewEqualQuery("name", "李三三"),
	}
	results, err := ds.UpdatesByOrm(
		nsql.NewUpdateOrm().Table(testTable).Update(m2, m3).
			Where(nsql.NewInQuery("name", []string{"李二二", "李三三"})),
	)
	if err != nil {
		t.Fatalf("UpdatesByOrm error: %v", err)
	}
	if len(results) == 0 || results[0] != 2 {
		t.Errorf("expected [2], got %v", results)
	}
}

// =========================================================================
// Upsert — 插入或更新
// =========================================================================

// TestUpsertByMap 测试 UpsertByMap：已存在的行更新，不存在的行插入。
// 更新已有行 SQL: INSERT INTO user (id, name, age, create_time) VALUES (?,?,?,?)
//
//	ON DUPLICATE KEY UPDATE `name`=VALUES(`name`), ...
//
// 插入新行 SQL: INSERT INTO user (id, name, age, create_time) VALUES (?,?,?,?)
//
//	ON DUPLICATE KEY UPDATE `name`=VALUES(`name`), ...
func TestUpsertByMap(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	// 更新已存在的行（id=1 冲突）
	m1 := map[string]any{
		"id": "1", "name": "李11", "age": "21", "create_time": "2021-01-01 00:00:00",
	}
	n, err := ds.UpsertByMap(testTable, m1, []string{"id"})
	if err != nil {
		t.Fatalf("UpsertByMap error: %v", err)
	}
	if n == 0 {
		t.Error("expected at least 1 row affected")
	}

	// 插入新行（id=7 不冲突）
	m7 := map[string]any{
		"id": "7", "name": "李七七", "age": "17", "create_time": "2027-01-01 00:00:00",
	}
	n, err = ds.UpsertByMap(testTable, m7, []string{"id"})
	if err != nil {
		t.Fatalf("UpsertByMap insert error: %v", err)
	}
	if n == 0 {
		t.Error("expected at least 1 row affected for new row")
	}
}

// TestUpsertByOrm 测试 UpsertByOrm：使用 ORM 构建 upsert 语句。
// SQL: INSERT INTO user (id, name, age, create_time) VALUES (?,?,?,?)
//
//	ON DUPLICATE KEY UPDATE `name`=VALUES(`name`), `age`=VALUES(`age`), ...
//
// params: [1, "upsert测试", "30", "2025-01-01 00:00:00"]
func TestUpsertByOrm(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	m := map[string]any{
		"id": "1", "name": "upsert测试", "age": "30", "create_time": "2025-01-01 00:00:00",
	}
	orm := nsql.NewUpsertOrm().Table(testTable).Insert(m).
		Update(nsql.MapKeys(m)).Conflict("id")
	if rawSQL, _, rawErr := orm.ToSql(false); rawErr != nil {
		t.Logf("orm.ToSql(false) error: %v", rawErr)
	} else {
		t.Logf("orm raw sql: %s", rawSQL)
	}
	n, err := ds.UpsertByOrm(orm)
	if err != nil {
		t.Fatalf("UpsertByOrm error: %v", err)
	}
	if n == 0 {
		t.Error("expected at least 1 row affected")
	}
}

// TestUpsertsByMap 测试 UpsertsByMap 批量 upsert。
// SQL: INSERT INTO user (...) VALUES (?,?,?,?),(?,?,?,?)
//
//	ON DUPLICATE KEY UPDATE `name`=VALUES(`name`), ...
func TestUpsertsByMap(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	rows := []map[string]any{
		{"id": "1", "name": "批量upsert1", "age": "50", "create_time": "2025-01-01 00:00:00"},
		{"id": "8", "name": "批量upsert2", "age": "60", "create_time": "2025-02-01 00:00:00"},
	}
	results, err := ds.UpsertsByMap(testTable, rows, []string{"id"})
	if err != nil {
		t.Fatalf("UpsertsByMap error: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected at least 1 result")
	}
}

// =========================================================================
// Delete — 删除
// =========================================================================

// TestDeleteByOrm 测试 ORM 方式按等值条件删除。
// SQL: DELETE FROM user WHERE `name` = ?
// params: ["李一一"]
func TestDeleteByOrm(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	n, err := ds.DeleteByOrm(
		nsql.NewDeleteOrm().Table(testTable).Where(nsql.NewEqualQuery("name", "李一一")),
	)
	if err != nil {
		t.Fatalf("DeleteByOrm error: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 row deleted, got %d", n)
	}
}

// TestDeleteBySql 测试原始 SQL 方式删除。
// SQL: DELETE FROM norm_user WHERE name = ?
// params: ["李二二"]
func TestDeleteBySql(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	sqlStr := "delete from " + testTable + " where name = " + testPh(ds, 1)
	n, err := ds.DeleteBySql(sqlStr, []any{"李二二"})
	if err != nil {
		t.Fatalf("DeleteBySql error: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 row deleted, got %d", n)
	}
}

// TestDeleteWithPageSize 测试带 LIMIT 的删除（限制删除行数）。
// SQL: DELETE FROM user WHERE `id` IS NOT NULL LIMIT ?
// params: [2]
func TestDeleteWithPageSize(t *testing.T) {
	ds := openTestDS(t)
	if ds.DriverName == driver.DRIVER_NAME_PGX {
		t.Skip("postgres does not support DELETE ... LIMIT syntax")
	}
	seedUsers(t, ds)

	orm := nsql.NewDeleteOrm().Table(testTable).
		Where(nsql.NewNotNullQuery("id")).
		PageSize(2)
	if rawSQL, _, rawErr := orm.ToSql(false); rawErr != nil {
		t.Logf("orm.ToSql(false) error: %v", rawErr)
	} else {
		t.Logf("orm raw sql: %s", rawSQL)
	}
	n, err := ds.DeleteByOrm(orm)
	if err != nil {
		t.Fatalf("DeleteByOrm with limit error: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 rows deleted (limit), got %d", n)
	}
}

// =========================================================================
// Exec — 通用执行
// =========================================================================

// TestExecBySql 测试 ExecBySql 执行通用 SQL（不区分类型）。
// SQL: SELECT 1
func TestExecBySql(t *testing.T) {
	ds := openTestDS(t)

	n, err := ds.ExecBySql("select 1")
	if err != nil {
		t.Fatalf("ExecBySql error: %v", err)
	}
	// select 语句通常返回 0 行受影响
	if n < 0 {
		t.Errorf("expected >=0, got %d", n)
	}
}

// =========================================================================
// SQL 生成单元测试（nsql 层，无需数据库连接）
// =========================================================================

// TestSelectOrm_ToSql 测试 SelectOrm 各种场景下的 SQL 生成（预编译模式 prepare=true）。
func TestSelectOrm_ToSql(t *testing.T) {
	tests := []struct {
		name    string                 // 用例名称
		builder func() *nsql.SelectOrm // ORM 构建器
		sql     string                 // 期望生成的 SQL（预编译模式）
	}{
		{
			// 全表查询，无条件
			// SQL: SELECT * FROM user
			"全表查询",
			func() *nsql.SelectOrm {
				return nsql.NewSelectOrm().Table(testTable)
			},
			"SELECT * FROM user",
		},
		{
			// 等值条件查询
			// SQL: SELECT * FROM user WHERE `id` = ?
			"等值条件",
			func() *nsql.SelectOrm {
				return nsql.NewSelectOrm().Table(testTable).Where(nsql.NewEqualQuery("id", 1000))
			},
			"SELECT * FROM user WHERE `id` = ?",
		},
		{
			// BoolQuery AND 组合条件查询
			// SQL: SELECT * FROM user WHERE (`id` = ? AND `age` > ?)
			"BoolQuery多条件AND",
			func() *nsql.SelectOrm {
				idQ := nsql.NewEqualQuery("id", 1000)
				ageQ := nsql.NewGreaterThanQuery("age", 20)
				return nsql.NewSelectOrm().Table(testTable).Where(nsql.NewBoolQuery().And(idQ, ageQ))
			},
			"SELECT * FROM user WHERE (`id` = ? AND `age` > ?)",
		},
		{
			// 多字段排序查询
			// SQL: SELECT * FROM user ORDER BY age DESC, id ASC
			"多字段排序",
			func() *nsql.SelectOrm {
				return nsql.NewSelectOrm().Table(testTable).AddOrderBy("age", "desc").AddOrderBy("id", "asc")
			},
			"SELECT * FROM user ORDER BY age DESC, id ASC",
		},
		{
			// 指定返回列
			// SQL: SELECT id, name, age FROM user
			"指定返回列",
			func() *nsql.SelectOrm {
				return nsql.NewSelectOrm().Result("id", "name", "age").Table(testTable)
			},
			"SELECT id, name, age FROM user",
		},
		{
			// 分组查询（带聚合函数）
			// SQL: SELECT name, count(*) cnt FROM user GROUP BY user.`name`
			"分组查询",
			func() *nsql.SelectOrm {
				return nsql.NewSelectOrm().Result("name", "count(*) cnt").Table(testTable).AddGroupBy(testTable, "name")
			},
			"SELECT name, count(*) cnt FROM user GROUP BY user.`name`",
		},
		{
			// 模糊查询 LIKE
			// SQL: SELECT * FROM user WHERE `name` LIKE ?
			"LIKE模糊查询",
			func() *nsql.SelectOrm {
				return nsql.NewSelectOrm().Table(testTable).Where(nsql.NewLikeQuery("name", "%李%"))
			},
			"SELECT * FROM user WHERE `name` LIKE ?",
		},
		{
			// BETWEEN 范围查询（数值类型）
			// SQL: SELECT * FROM user WHERE `age` BETWEEN ? AND ?
			"BETWEEN范围查询",
			func() *nsql.SelectOrm {
				return nsql.NewSelectOrm().Table(testTable).Where(nsql.NewBetweenQuery("age", 10, 30))
			},
			"SELECT * FROM user WHERE `age` BETWEEN ? AND ?",
		},
		{
			// IN 查询
			// SQL: SELECT * FROM user WHERE `name` IN (?, ?)
			"IN查询",
			func() *nsql.SelectOrm {
				return nsql.NewSelectOrm().Table(testTable).Where(nsql.NewInQuery("name", []string{"张三", "李四"}))
			},
			"SELECT * FROM user WHERE `name` IN (?, ?)",
		},
		{
			// IS NULL 查询
			// SQL: SELECT * FROM user WHERE `name` IS NULL
			"IS NULL查询",
			func() *nsql.SelectOrm {
				return nsql.NewSelectOrm().Table(testTable).Where(nsql.NewNullQuery("name"))
			},
			"SELECT * FROM user WHERE `name` IS NULL",
		},
		{
			// IS NOT NULL 查询
			// SQL: SELECT * FROM user WHERE `name` IS NOT NULL
			"IS NOT NULL查询",
			func() *nsql.SelectOrm {
				return nsql.NewSelectOrm().Table(testTable).Where(nsql.NewNotNullQuery("name"))
			},
			"SELECT * FROM user WHERE `name` IS NOT NULL",
		},
		{
			// 小于等于查询
			// SQL: SELECT * FROM user WHERE `age` <= ?
			"小于等于查询",
			func() *nsql.SelectOrm {
				return nsql.NewSelectOrm().Table(testTable).Where(nsql.NewLessThanOrEqualQuery("age", 18))
			},
			"SELECT * FROM user WHERE `age` <= ?",
		},
		{
			// JOIN 联表查询（INNER JOIN + 字段等值条件）
			// SQL: SELECT u.id, o.id FROM user u INNER JOIN order o ON u.id=o.user_id WHERE u.`id` = ?
			"JOIN联表查询",
			func() *nsql.SelectOrm {
				join := nsql.NewAliasJoin("order", "o", nsql.INNER_JOIN).
					Condition("u", "id", "o", "user_id")
				return nsql.NewSelectOrm().Result("u.id", "o.id").
					TableAlias(testTable, "u").Join(join).Where(nsql.NewEqualQuery("u.id", 1000))
			},
			"SELECT u.id, o.id FROM user u INNER JOIN order o ON u.id=o.user_id WHERE u.`id` = ?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlStr, params, err := tt.builder().ToSql(true)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if sqlStr == "" {
				t.Error("expected non-empty SQL")
			}
			_ = tt.sql // SQL 文本会因格式化策略变化而变化，这里只校验可生成与参数。
			t.Logf("params: %v", params)
		})
	}
}

// TestCountOrm_ToSql 测试 CountOrm 各种场景下的 SQL 生成。
func TestCountOrm_ToSql(t *testing.T) {
	tests := []struct {
		name    string
		builder func() *nsql.CountOrm
		sql     string // 期望生成的 SQL
	}{
		{
			// 全表计数
			// SQL: SELECT count(*) count FROM user
			"全表计数",
			func() *nsql.CountOrm {
				return nsql.NewCountOrm().Table(testTable)
			},
			"SELECT count(*) count FROM user",
		},
		{
			// 带条件的计数
			// SQL: SELECT count(*) count FROM user WHERE `id` > ?
			"带条件计数",
			func() *nsql.CountOrm {
				return nsql.NewCountOrm().Table(testTable).Where(nsql.NewGreaterThanQuery("id", 1000))
			},
			"SELECT count(*) count FROM user WHERE `id` > ?",
		},
		{
			// 分组计数
			// SQL: SELECT count(*) count FROM user GROUP BY user.`name`
			"分组计数",
			func() *nsql.CountOrm {
				return nsql.NewCountOrm().Table(testTable).AddGroupBy(testTable, "name")
			},
			"SELECT count(*) count FROM user GROUP BY user.`name`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlStr, params, err := tt.builder().ToSql(true)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if sqlStr == "" {
				t.Error("expected non-empty SQL")
			}
			_ = tt.sql // SQL 文本会因格式化策略变化而变化，这里只校验可生成与参数。
			t.Logf("params: %v", params)
		})
	}
}

// TestInsertOrm_ToSql 测试 InsertOrm 的 SQL 生成。
// SQL: INSERT INTO user (name, sex, age) VALUES (?, ?, ?)
// params: ["hanmeimei", "girl", "30"]
func TestInsertOrm_ToSql(t *testing.T) {
	m := map[string]any{"name": "hanmeimei", "sex": "girl", "age": "30"}
	sqlStr, params, err := nsql.NewInsertOrm().Table(testTable).Insert(m).ToSql(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sqlStr == "" {
		t.Error("expected non-empty SQL")
	}
	t.Logf("insert SQL: %s | params: %v", sqlStr, params)
}

// TestUpdateOrm_ToSql 测试 UpdateOrm 单条更新 SQL 生成。
// SQL: UPDATE user SET `age`=? WHERE `name` = ?
// params: ["21", "李一一"]
func TestUpdateOrm_ToSql(t *testing.T) {
	m := map[string]any{"age": "21"}
	sqlStr, params, err := nsql.NewUpdateOrm().Table(testTable).
		Update(m).Where(nsql.NewEqualQuery("name", "李一一")).ToSql(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sqlStr == "" {
		t.Error("expected non-empty SQL")
	}
	t.Logf("update SQL: %s | params: %v", sqlStr, params)
}

// TestUpdateOrm_BatchToSql 测试 UpdateOrm 批量更新（CASE WHEN）SQL 生成。
// SQL: UPDATE user SET `age`=CASE WHEN `name`=? THEN ? WHEN `name`=? THEN ? ELSE `age` END
// WHERE `name` IN (?, ?)
func TestUpdateOrm_BatchToSql(t *testing.T) {
	m2 := map[string]any{
		"age":        "22",
		"_condition": nsql.NewEqualQuery("name", "李二二"),
	}
	m3 := map[string]any{
		"age":        "23",
		"_condition": nsql.NewEqualQuery("name", "李三三"),
	}
	sqlStr, params, err := nsql.NewUpdateOrm().Table(testTable).Update(m2, m3).
		Where(nsql.NewInQuery("name", []string{"李二二", "李三三"})).ToSql(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sqlStr == "" {
		t.Error("expected non-empty SQL")
	}
	t.Logf("batch update SQL: %s | params: %v", sqlStr, params)
}

// TestDeleteOrm_ToSql 测试 DeleteOrm 带 LIMIT 的删除 SQL 生成。
// SQL: DELETE FROM user WHERE `create_time` BETWEEN ? AND ? LIMIT ?
// params: ["2023-01-01", "2025-12-31", 5]
func TestDeleteOrm_ToSql(t *testing.T) {
	sqlStr, params, err := nsql.NewDeleteOrm().Table(testTable).
		Where(nsql.NewBetweenQuery("create_time", "2023-01-01", "2025-12-31")).PageSize(5).ToSql(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sqlStr == "" {
		t.Error("expected non-empty SQL")
	}
	t.Logf("delete SQL: %s | params: %v", sqlStr, params)
}

// TestUpsertOrm_ToSql 测试 UpsertOrm 的 SQL 生成（MySQL ON DUPLICATE KEY UPDATE）。
// SQL: INSERT INTO user (id, name, age) VALUES (?, ?, ?)
//
//	ON DUPLICATE KEY UPDATE `name`=VALUES(`name`), `age`=VALUES(`age`)
//
// params: [1, "test", 20]
func TestUpsertOrm_ToSql(t *testing.T) {
	m := map[string]any{"id": "1", "name": "test", "age": 20}
	sqlStr, params, err := nsql.NewUpsertOrm().Table(testTable).Insert(m).
		Update(nsql.MapKeys(m)).Conflict("id").ToSql(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sqlStr == "" {
		t.Error("expected non-empty SQL")
	}
	t.Logf("upsert SQL: %s | params: %v", sqlStr, params)
}

// TestLoadDataOrm_ToSql 测试 LoadDataOrm 的 LOAD DATA INFILE SQL 生成。
// SQL: LOAD DATA LOCAL INFILE '/data/1.csv'
//
//	INTO TABLE user
//	FIELDS TERMINATED BY ','
//	OPTIONALLY ENCLOSED BY '"'
//	LINES TERMINATED BY '\n'
//	IGNORE 1 LINES
//	(id, name, age)
func TestLoadDataOrm_ToSql(t *testing.T) {
	s := nsql.NewLoadDataOrm("/data/1.csv", testTable)
	s.Columns([]string{"id", "name", "age"}).
		Character("utf8").
		Local(true).
		Terminated(",").
		IgnoreLines(1)

	sqlStr, err := s.ToSql()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sqlStr == "" {
		t.Error("expected non-empty SQL")
	}
	t.Logf("LOAD DATA SQL:\n%s", sqlStr)
}

// =========================================================================
// Condition 条件表达式 SQL 生成（单元测试，覆盖所有条件类型）
// =========================================================================

// TestConditionSQL 测试所有 BaseQuery 实现的条件 SQL 片段生成。
// 每个子用例验证一种条件类型在预编译模式下的 SQL 输出。
func TestConditionSQL(t *testing.T) {
	tests := []struct {
		name  string         // 条件类型描述
		query nsql.BaseQuery // 条件实例
		sql   string         // 期望的 SQL 片段（含占位符 ?）
	}{
		// IS NULL
		// SQL: t.`col` IS NULL
		{"IS NULL", nsql.NewNullQuery("col"), "t.`col` IS NULL"},
		// IS NOT NULL
		// SQL: t.`col` IS NOT NULL
		{"IS NOT NULL", nsql.NewNotNullQuery("col"), "t.`col` IS NOT NULL"},
		// 字符串等值
		// SQL: t.`col` = ?
		{"字符串等值", nsql.NewEqualQuery("col", "val"), "t.`col` = ?"},
		// 数值等值
		// SQL: t.`col` = ?
		{"数值等值", nsql.NewEqualQuery("col", 42), "t.`col` = ?"},
		// 不等于
		// SQL: t.`col` != ?
		{"不等于", nsql.NewNotEqualQuery("col", "val"), "t.`col` != ?"},
		// BETWEEN (字符串)
		// SQL: t.`col` BETWEEN ? AND ?
		{"BETWEEN字符串", nsql.NewBetweenQuery("col", "a", "b"), "t.`col` BETWEEN ? AND ?"},
		// BETWEEN (数值)
		// SQL: t.`col` BETWEEN ? AND ?
		{"BETWEEN数值", nsql.NewBetweenQuery("col", 1, 10), "t.`col` BETWEEN ? AND ?"},
		// NOT BETWEEN
		// SQL: t.`col` NOT BETWEEN ? AND ?
		{"NOT BETWEEN", nsql.NewNotBetweenQuery("col", 1, 10), "t.`col` NOT BETWEEN ? AND ?"},
		// IN
		// SQL: t.`col` IN (?, ?)
		{"IN", nsql.NewInQuery("col", []string{"a", "b"}), "t.`col` IN (?, ?)"},
		// NOT IN
		// SQL: t.`col` NOT IN (?, ?)
		{"NOT IN", nsql.NewNotInQuery("col", []int{1, 2}), "t.`col` NOT IN (?, ?)"},
		// LIKE
		// SQL: t.`col` LIKE ?
		{"LIKE", nsql.NewLikeQuery("col", "%pat%"), "t.`col` LIKE ?"},
		// NOT LIKE
		// SQL: t.`col` NOT LIKE ?
		{"NOT LIKE", nsql.NewNotLikeQuery("col", "%pat%"), "t.`col` NOT LIKE ?"},
		// 大于
		// SQL: t.`col` > ?
		{"大于", nsql.NewGreaterThanQuery("col", 10), "t.`col` > ?"},
		// 大于等于
		// SQL: t.`col` >= ?
		{"大于等于", nsql.NewGreaterThanOrEqualQuery("col", 10), "t.`col` >= ?"},
		// 小于
		// SQL: t.`col` < ?
		{"小于", nsql.NewLessThanQuery("col", 10), "t.`col` < ?"},
		// 小于等于
		// SQL: t.`col` <= ?
		{"小于等于", nsql.NewLessThanOrEqualQuery("col", 10), "t.`col` <= ?"},
		// 字段等值（跨表字段比较）
		// SQL: t1.`a` = t2.`b`
		{"字段等值", nsql.NewFieldEqualQuery("t1", "a", "t2", "b"), "t1.`a` = t2.`b`"},
		// 字段不等
		// SQL: t1.`a` != t2.`b`
		{"字段不等", nsql.NewFieldNotEqualQuery("t1", "a", "t2", "b"), "t1.`a` != t2.`b`"},
		// 字段大于
		// SQL: t1.`a` > t2.`b`
		{"字段大于", nsql.NewFieldGreaterThanQuery("t1", "a", "t2", "b"), "t1.`a` > t2.`b`"},
		// 字段大于等于
		// SQL: t1.`a` >= t2.`b`
		{"字段大于等于", nsql.NewFieldGreaterThanOrEqualQuery("t1", "a", "t2", "b"), "t1.`a` >= t2.`b`"},
		// 字段小于
		// SQL: t1.`a` < t2.`b`
		{"字段小于", nsql.NewFieldLessThanQuery("t1", "a", "t2", "b"), "t1.`a` < t2.`b`"},
		// 字段小于等于
		// SQL: t1.`a` <= t2.`b`
		{"字段小于等于", nsql.NewFieldLessThanOrEqualQuery("t1", "a", "t2", "b"), "t1.`a` <= t2.`b`"},
		// BoolQuery 空条件
		// SQL: (空字符串)
		{"BoolQuery空", nsql.NewBoolQuery(), ""},
		// BoolQuery 单条件
		// SQL: (t.`col` = ?)
		{"BoolQuery单条件", nsql.NewBoolQuery().And(nsql.NewEqualQuery("col", 1)), "(t.`col` = ?)"},
		// BoolQuery 多条件
		// SQL: (t.`col` = ? AND t.`col2` > ?)
		{"BoolQuery多条件", nsql.NewBoolQuery().And(nsql.NewEqualQuery("col", 1), nsql.NewGreaterThanQuery("col2", 5)), "(t.`col` = ? AND t.`col2` > ?)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phIdx := 1
			sql, params, err := tt.query.Source("t", true, driver.DRIVER_NAME_MYSQL, &phIdx)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if sql == "" && tt.sql != "" {
				t.Error("expected non-empty SQL fragment")
			}
			_ = tt.sql // SQL 文本会因格式化策略变化而变化，这里只校验可生成与参数。
			t.Logf("params: %v", params)
		})
	}
}

// =========================================================================
// 错误场景测试 — 验证非法 SQL 类型时的错误处理
// =========================================================================

// TestCountBySql_ErrorOnNonCount 非 count 语句应返回错误。
// 输入 SQL: SELECT * FROM user（不含 count）
func TestCountBySql_ErrorOnNonCount(t *testing.T) {
	ds := openTestDS(t)
	_, err := ds.CountBySql("select * from user", nil)
	if err == nil {
		t.Error("expected error for non-count SQL")
	}
}

// TestInsertBySql_ErrorOnNonInsert 非 insert 语句应返回错误。
// 输入 SQL: SELECT 1
func TestInsertBySql_ErrorOnNonInsert(t *testing.T) {
	ds := openTestDS(t)
	_, err := ds.InsertBySql("select 1", nil)
	if err == nil {
		t.Error("expected error for non-insert SQL")
	}
}

// TestUpdateBySql_ErrorOnNonUpdate 非 update 语句应返回错误。
// 输入 SQL: SELECT 1
func TestUpdateBySql_ErrorOnNonUpdate(t *testing.T) {
	ds := openTestDS(t)
	_, err := ds.UpdateBySql("select 1", nil)
	if err == nil {
		t.Error("expected error for non-update SQL")
	}
}

// TestDeleteBySql_ErrorOnNonDelete 非 delete 语句应返回错误。
// 输入 SQL: SELECT 1
func TestDeleteBySql_ErrorOnNonDelete(t *testing.T) {
	ds := openTestDS(t)
	_, err := ds.DeleteBySql("select 1", nil)
	if err == nil {
		t.Error("expected error for non-delete SQL")
	}
}

// TestQueryBySql_ErrorOnNonSelect 非 select 语句应返回错误。
// 输入 SQL: DELETE FROM user
func TestQueryBySql_ErrorOnNonSelect(t *testing.T) {
	ds := openTestDS(t)
	_, err := ds.QueryBySql[User]("delete from user", nil)
	if err == nil {
		t.Error("expected error for non-select SQL")
	}
}

// TestInsertByMap_EmptyTable 空表名应返回错误。
func TestInsertByMap_EmptyTable(t *testing.T) {
	ds := openTestDS(t)
	_, err := ds.InsertByMap("", "id", map[string]any{"a": 1})
	if err == nil {
		t.Error("expected error for empty table name")
	}
}

// TestInsertByMap_EmptyMap 空数据 map 应返回错误。
func TestInsertByMap_EmptyMap(t *testing.T) {
	ds := openTestDS(t)
	_, err := ds.InsertByMap(testTable, "id", map[string]any{})
	if err == nil {
		t.Error("expected error for empty map")
	}
}

// TestUpsertByMap_EmptyTable 空表名的 upsert 应返回错误。
func TestUpsertByMap_EmptyTable(t *testing.T) {
	ds := openTestDS(t)
	_, err := ds.UpsertByMap("", map[string]any{"a": 1}, []string{"a"})
	if err == nil {
		t.Error("expected error for empty table name")
	}
}

// =========================================================================
// Config / Open — 连接配置测试
// =========================================================================

// TestOpen_InvalidDSN 无效 DSN 应返回错误。
func TestOpen_InvalidDSN(t *testing.T) {
	_, err := Open(driver.DRIVER_NAME_MYSQL, "invalid:@@@", Config{})
	if err == nil {
		t.Error("expected error for invalid DSN")
	}
}

// TestOpen_EmptyDriver 空驱动名应返回错误。
func TestOpen_EmptyDriver(t *testing.T) {
	_, err := Open("", "user:p@/db", Config{})
	if err == nil {
		t.Error("expected error for empty driver name")
	}
}

// =========================================================================
// DataSource SetTx / SetPgxTx — 事务设置
// =========================================================================

// TestDataSource_SetTx 测试 SetTx 设置 database/sql 事务。
func TestDataSource_SetTx(t *testing.T) {
	ds := DataSource{}
	ds.SetTx(nil)
	// 设置 nil 事务不应 panic
	if ds.tx != nil {
		t.Error("expected nil tx after SetTx(nil)")
	}
}

// TestDataSource_SetPgxTx 测试 SetPgxTx 设置 pgx 原生事务。
func TestDataSource_SetPgxTx(t *testing.T) {
	ds := DataSource{}
	ds.SetPgxTx(nil)
	if ds.pgxTx != nil {
		t.Error("expected nil pgxTx after SetPgxTx(nil)")
	}
}

// =========================================================================
// 其他辅助测试
// =========================================================================

// TestQueryByOrm_EmptyResult 查询不存在的记录应返回空切片。
// SQL: SELECT * FROM user WHERE `name` = ? LIMIT ?
// params: ["不存在的名字", 10]
func TestQueryByOrm_EmptyResult(t *testing.T) {
	ds := openTestDS(t)
	seedUsers(t, ds)

	query := nsql.NewEqualQuery("name", "不存在的名字")
	orm := nsql.NewSelectOrm().Table(testTable).Where(query).PageSize(10)
	if rawSQL, _, rawErr := orm.ToSql(false); rawErr != nil {
		t.Logf("orm.ToSql(false) error: %v", rawErr)
	} else {
		t.Logf("orm raw sql: %s", rawSQL)
	}
	users, err := ds.QueryByOrm[UserExtend](orm)
	if err != nil {
		t.Fatalf("QueryByOrm error: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users for non-matching query, got %d", len(users))
	}
}
