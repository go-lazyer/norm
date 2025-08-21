

# north

### 一、介绍

#### 1、安装

```
go get github.com/go-lazyer/norm@latest
```
#### 2、介绍

north 由3部分组成。north为主模块，用与真实与数据库交互。nsql`github.com/go-lazyer/norm/nsql`和ngen `github.com/go-lazyer/norm/ngen` 为辅助模块，其中nsql用于sql 生成，ngen用于生成表对应的struct，以及一些相关dao层代码，减少开发者自行拼接sql，和table to struct的工作量，可以专注于核心业务代码。该工具是模仿 mybatis-generator 实现的，同时又借鉴了go elastic工具的包github.com/olivere/elastic 的查询实现方式，所以熟悉 mybatis-generator和olivere 及容易上手。该工具支持mysql/pgsql。

#### 3、简单查询

```go
func (){
    type User struct {
      Id           sql.NullString `orm:"id" `                   //
      Nickname     sql.NullString `orm:"nickname" default:""`   // 昵称
      Username     sql.NullString `orm:"username" `             // 用户名
    }	

    dsn := "user:password@tcp(127.0.0.1:3306)/dbname"
    db, err := north.Open("mysql", args, north.Config{})
    if err != nil {
        return nil, errors.WithStack(err)
    }

  	//select * from user where id='123456'
    orm := nsql.NewSelectOrm().Table("user").Where(nsql.NewEqualQuery(id, "123456"))

    sqlStr, params, err := orm.ToSql(true)

    users, err := north.PrepareQuery[User](sqlStr, params, db)
    if err != nil {
        return nil, errors.WithStack(err)
    }
}
```



### 二、north

north提供了以下方法，用于增删改查

```go
//普通count
func Count(sqlStr string, params []any, ds DataSource) (int64, error)
//预处理count
func PrepareCount(sqlStr string, params [][]any, ds DataSource) ([]int64, error)

//普通查询
func Query[T any](sqlStr string, params []any, ds DataSource) ([]T, error)
//预处理查询
func PrepareQuery[T any](sqlStr string, params [][]any, ds DataSource) ([][]T, error)

//插入
func Insert(sqlStr string, params []any, ds DataSource) (int64, error)
//预处理插入
func PrepareInsert(sqlStr string, params [][]any, ds DataSource) ([]int64, error)

//修改
func Update(sqlStr string, params []any, ds DataSource) (int64, error)
//预处理修改
func PrepareUpdate(sqlStr string, params [][]any, ds DataSource) ([]int64, error) 

//删除
func Delete(sqlStr string, params []any, ds DataSource) (int64, error)
//预处理删除
func PrepareDelete(sqlStr string, params [][]any, ds DataSource) ([]int64, error)
```



### 三、nsql 生成sql

nsql 可以生成普通sql和预处理sql，配合north模块可以轻松实现增删改查.

#### 1、引入

```go
import  "github.com/go-lazyer/norm/nsql"
```



#### 2、统计

``` go
//select count(1) count from user where t.id>1000

query := nsql.NewGreaterThanQuery("id", 1000)

gen := nsql.NewCountOrm().Table("user").Where(query)

fmt.Println(gen.ToSql(false))
  
```
#### 3、基础查询

```go
//select * from user

orm := nsql.NewOrm().Table("user")

fmt.Println(orm.ToSql(false))

//select * from user where t.id=1000

query := nsql.NewEqualQuery("id", 1000)

orm := nsql.NewSelectOrm().Table("user").Where(query)

fmt.Println(orm.ToSql(false))
```

#### 4、排序查询

```go
// select * from user where id=1000 and age>20 order by age desc,id asc

idQuery := nsql.NewEqualQuery("id", 1000)

ageQuery := nsql.NewGreaterThanQuery("age", 20)

boolQuery := nsql.NewBoolQuery().And(idQuery, ageQuery)

orm := nsql.NewSelectOrm().Table("user").Where(boolQuery).AddOrderBy("age", "desc").AddOrderBy("id", "asc")

fmt.Println(orm.ToSql(false))
```

#### 5、复杂查询

```go
// select id,name,age from user where (id=1000 and age>20) or age <=10 order by age desc

idQuery := nsql.NewEqualQuery("id", 1000)

ageQuery := nsql.NewGreaterThanQuery("age", 20)

boolQuery := nsql.NewBoolQuery().And(idQuery, ageQuery)

ageQuery2 := nsql.NewLessThanOrEqualQuery("age", 10)

orm := nsql.NewSelectOrm().Result("id", "name", "age").Table("user").Where(boolQuery, ageQuery2).AddOrderBy("age", "desc")

fmt.Println(orm.ToSql(false))
```

#### 6、联表查询

 ```go
 // select user.id,order.id  from user join order on user.id=order.user_id where user.id='10000'
 
 idQuery = nsql.NewEqualQuery("id", 1000)
 
 join := nsql.NewJoin("order", INNER_JOIN).Condition("user", "id", "order", "user_id")
 
 orm = nsql.NewSelectOrm().Result("user.id", "order.id").Table("user").Join(join).Where(idQuery)
 
 fmt.Println(orm.ToSql(false))
 ```

#### 7、更新

```go
// update user set age=21,name="lazeyr" where id="10000"	

query := nsql.NewEqualQuery("id", 1000)
set := map[string]any{
  "age":  21,
  "name": "lazyer",
}

gen := nsql.NewUpdateOrm().Table("user").Where(query).Update(set)
fmt.Println(gen.ToSql(false))
```

#### 8、批量更新

```go
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
	// 	user.create_time ='2025-01-01 00:00:00'
func TestUpdatesSql(t *testing.T) {
	f1 := map[string]any{
		"sex":        "boy",
		"age":        "20",
		"_condition": nsql.NewEqualQuery("dwid", "10001"),
	}
	f2 := map[string]any{
		"sex": "girl",
		// "age":        "40",
		"_condition": nsql.NewEqualQuery("name", "lilie"),
	}

	query := nsql.NewEqualQuery("create_time", "2025-01-01 00:00:00")
	gen := nsql.NewUpdateOrm().Table("user").Where(query).Update(f1, f2)
	fmt.Println(gen.ToSql(false))
}
```

#### 9、单条插入

```go
// insert into `user` ( age , name , sex ) values ( '10' , 'lilie' , 'boy' ),

m := map[string]any{
  "name": "lilie",
  "sex":  "boy",
  "age":  "10",
}

gen := nsql.NewInsertOrm().Table(model.TABLE_NAME).Insert(m)

fmt.Println(gen.ToSql(false))
```

#### 10、批量插入

```go
//insert into `user` ( age , name , sex ) 
//values
//( '10' , 'lilie' , 'boy' ),
//( '20' , 'lining' , 'boy' ),
//( '30' , 'hanmeimei' , 'girl' )
f1 := map[string]any{
  "name": "lilie",
  "sex":  "boy",
  "age":  "10",
}
f2 := map[string]any{
  "name": "lining",
  "sex":  "boy",
  "age":  "20",
}
f3 := map[string]any{
  "name": "hanmeimei",
  "sex":  "girl",
  "age":  "30",
}
orm := nsql.NewInsertOrm().Table("user").Insert(f1,f2,f3)

fmt.Print(orm.ToSql(false))
```

### 四、ngen生成代码

ngen 模块主要用于生成数据库表对应的struct，以及dao文件，同时会生成相关的附属类文件

#### 文件介绍

1. model 文件,struct 所在文件，每次都会更新。
2. extend文件，model的扩展文件，用于接收联表查询的返回值，只生成一次。
3. view文件，提供接口时，接口中的返回值，和model 独立，只生成一次。
4. param文件，提供接口时，用于接收接口的参数，只生成一次。
5. dao文件，提供常用的增删改查方法，只生成一次。

#### 使用教程

新建main方法，配置需要生成代码的表(支持配置多个)，运行即可生成代码

```go
package main

import (
	ngen "github.com/go-lazyer/norm/ngen"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dsn := "root:123@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=true&loc=Local"
	var moduls = []ngen.Module{
		{//最小配置
		 	TableName:  "user",
		 	ModulePath: "/Users/Lazyer/Workspace/lazyer/api/user",
		},
    { //完整配置
			TableName:             "user",                                 //表名
			ModulePath:            "/Users/Lazyer/Workspace/test/api/user", //相对路径，包含项目名
			Model:                 true,                                   //是否生成Model层代码
			ModelPackageName:      "model",                                //Model层包名
			ModelFileName:         "user_model.go",                        //Model层文件名
			Extend:                true,                                   //是否生成层代码
			ExtendPackageName:     "extend",                               //Extend包名
			ExtendFileName:        "user_extend.go",                       //Extend文件名
			Param:                 true,                                   //是否生成Param代码
			ParamPackageName:      "param",                                //Param包名
			ParamFileName:         "user_param.go",                        //Param文件名
			View:                  true,                                   //是否生成View代码
			ViewPackageName:       "view",                                 //View包名
			ViewFileName:          "user_view.go",                         //View文件名
			Dao:                   true,                                   //是否生成Dao代码
			DaoPackageName:        "dao",                                  //Dao层包名
			DaoFileName:           "user_dao.go",                          //Dao层文件名
			Service:               true,                                   //是否生成Service层代码
			ServicePackageName:    "service",                              //Service层包名
			ServiceFileName:       "user_service.go",                      //Service层文件名
			Controller:            true,                                   //是否生成Controller层代码
			ControllerPackageName: "controller",                           //Controller层包名
			ControllerFileName:    "user_controller.go",                   //Controller层文件名
		},
	}
	ngen.NewGen().Dsn(dsn).Project("test").Gen(moduls)
}
```

