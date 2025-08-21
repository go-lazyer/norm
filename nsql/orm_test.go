package nsql

import (
	"database/sql"
	"fmt"
	"testing"
	// _ "github.com/go-sql-driver/mysql"
)

const (
	ID  = "Id"  //
	DAY = "day" //
	NUM = "num" //
)

type Test struct {
	Id sql.NullString `orm:"id" ` //
}
type Test1 struct {
	Test
	Num sql.NullInt64 `orm:"num" ` //
}

func TestGenerator_ToSql(t *testing.T) {
	// select count(1) count from user where t.id>1000
	query := NewGreaterThanQuery("id", 1000)
	gen := NewCountOrm().Table("user").Where(query)
	fmt.Println(gen.ToSql(false))
}

func TestGenerator_GroupSql(t *testing.T) {
	//select count(1) count from user where t.id>1000
	query := NewGreaterThanQuery("id", 1000)
	gen := NewCountOrm().Table("user").Where(query)
	fmt.Println(gen.ToSql(false))
}

func TestGenerator_ToSql1(t *testing.T) {
	//select * from user
	query1 := NewBoolQuery()
	gen := NewSelectOrm().Table("user").Where(query1)
	fmt.Println(gen.ToSql(false))
}
func TestGenerator_ToSql2(t *testing.T) {
	//select * from user where t.id=1000
	query := NewEqualQuery("id", 1000)
	gen := NewSelectOrm().Table("user").Where(query)
	fmt.Println(gen.ToSql(false))
}
func TestGenerator_ToSql3(t *testing.T) {
	// select * from user where id=1000 and age>20 order by age desc
	idQuery := NewEqualQuery("id", 1000)
	ageQuery := NewGreaterThanQuery("age", 20)
	boolQuery := NewBoolQuery().And(idQuery, ageQuery)
	gen := NewSelectOrm().Table("user").Where(boolQuery).AddOrderBy("age", "desc")
	fmt.Println(gen.ToSql(false))
}
func TestGenerator_ToSql4(t *testing.T) {
	// select id,name,age from user where (id=1000 and age>20) or age <=10 order by age desc ,id asc
	idQuery := NewEqualQuery("id", 1000)
	ageQuery := NewGreaterThanQuery("age", 20)
	boolQuery := NewBoolQuery().And(idQuery, ageQuery)
	ageQuery2 := NewLessThanOrEqualQuery("age", 10)
	gen := NewSelectOrm().Result("id", "name", "age").Table("user").Where(boolQuery, ageQuery2).AddOrderBy("age", "desc").AddOrderBy("id", "asc")
	fmt.Println(gen.ToSql(false))
}
func TestGenerator_ToSql5(t *testing.T) {
	// select user.id,order.id  from user join order on user.id=order.user_id where user.id='10000'
	idQuery := NewEqualQuery("id", 1000)

	join := NewAliasJoin("order", "o", INNER_JOIN).Condition("u", "id", "o", "user_id")
	gen := NewSelectOrm().Result("u.id", "o.id").TableAlias("user", "u").Join(join).Where(idQuery)
	fmt.Println(gen.ToSql(false))
}

func TestGenerator_ToSql6(t *testing.T) {
	// select user.sex,count(user.sex) count  from user group by user.sex

	gen := NewSelectOrm().Result("user.sex", "count(user.sex) count").Table("user").AddGroupBy("user", "sex")
	fmt.Println(gen.ToSql(false))
}

func TestGenerator_ToSql7(t *testing.T) {
	// select user.id,order.id  from user join order on user.id=order.user_id and order.create_time>user.create_time where user.id='10000'
	idQuery := NewEqualQuery("id", 1000)

	join := NewAliasJoin("order", "o", INNER_JOIN).Condition("u", "id", "o", "user_id").Where(NewFieldGreaterThanQuery("o", "create_time", "u", "create_time"))
	gen := NewSelectOrm().Result("u.id", "o.id").TableAlias("user", "u").Join(join).Where(idQuery)
	fmt.Println(gen.ToSql(true))
}

func TestGenerator_InsertSql(t *testing.T) {
	f3 := map[string]any{
		"name": "hanmeimei",
		"sex":  "girl",
		"age":  "30",
	}
	gen := NewInsertOrm().Table("user").Insert(f3)
	fmt.Print(gen.ToSql(false))
}
func TestGenerator_InsertsSql(t *testing.T) {
	//insert into `user` ( age , name , sex ) values( '10' , 'lilie' , 'boy' ),
	//( '20' , 'lining' , 'boy' ),
	//( '30' , 'hanmeimei' , 'girl' )
	f1 := map[string]any{
		"sex":  "boy",
		"name": "lilie",
		"age":  "10",
	}
	f2 := map[string]any{
		"name": "lining",
		"age":  "20",
		"sex":  "boy",
	}
	f3 := map[string]any{
		"name": "hanmeimei",
		"sex":  "girl",
		"age":  "30",
	}
	gen := NewInsertOrm().Table("user").Insert(f1, f2, f3)
	fmt.Print(gen.ToSql(false))
}
func TestGenerator_UpdateSql(t *testing.T) {
	// update user set age=21,name="lazeyr" where id="10000"
	query := NewEqualQuery("id", 1000)
	set := map[string]any{
		"age":  21,
		"name": "lazyer",
	}
	gen := NewUpdateOrm().Table("user").Where(query).Update(set)
	fmt.Println(gen.ToSql(false))
}

func TestGenerator_UpdatesSql(t *testing.T) {
	// update
	// `user`
	// set
	// 	sex = case
	// 	when dwid=10001 then boy
	// 	when name='lilie' then girl
	//  else sex
	// 	end,
	// 	age = case dwid
	// 	when dwid=10001 then 20
	// 	when name='lilie' then 40
	//  else age
	// 	end
	// where
	// 	user.create_time >'2025-01-01 00:00:00'
	f1 := map[string]any{
		"sex":        "boy",
		"age":        "20",
		"_condition": NewEqualQuery("dwid", "10001"),
	}
	f2 := map[string]any{
		"sex": "girl",
		// "age":        "40",
		"_condition": NewEqualQuery("name", "lilie"),
	}

	query := NewEqualQuery("create_time", "2025-01-01 00:00:00")
	gen := NewUpdateOrm().Table("user").Where(query).Update(f1, f2)
	fmt.Println(gen.ToSql(false))
}

func TestGenerator_UpsertSql(t *testing.T) {
	// insert into user ( name , id , age ) values( 'lazyer' , '1' , '21' ) on duplicate key update name='lazyer',age='21'
	set := map[string]any{
		"id":   1,
		"age":  21,
		"name": "lazyer",
	}

	upMap := []string{"age", "name"}

	gen := NewUpsertOrm().Table("user").Insert(set).Update(upMap)
	fmt.Println(gen.ToSql(false))
}

func TestGenerator_UpsertsSql(t *testing.T) {

	set := map[string]any{
		"id":   1,
		"age":  21,
		"name": "lazyer",
	}
	set1 := map[string]any{
		"id":   2,
		"age":  211,
		"name": "lazyer1",
	}

	upMap := []string{"age", "name"}

	gen := NewUpsertOrm().Table("user").Insert(set, set1).Update(upMap)
	fmt.Println(gen.ToSql(false))
}
func TestGenerator_PrepareUpsertSql(t *testing.T) {

	set := map[string]any{
		"id":   1,
		"age":  21,
		"name": "lazyer",
	}
	set1 := map[string]any{
		"id":   2,
		"age":  211,
		"name": "lazyer1",
	}

	upMap := []string{"age", "name"}

	gen := NewUpsertOrm().Table("user").Insert(set, set1).Update(upMap)
	fmt.Println(gen.ToPrepareSql())
}
