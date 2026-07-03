# ngen 多数据库支持设计方案

> 版本: v1.0 | 日期: 2026-05-22

## 1. 背景与目标

### 1.1 现状

`ngen_new/gen.go` 是 norm ORM 的代码生成器，通过读取数据库元数据（表结构、字段信息），自动生成 model/dao/service/controller 等脚手架代码。当前已支持 MySQL 和 PostgreSQL。

**当前架构的核心问题：**

| 问题 | 位置 | 说明 |
|------|------|------|
| 硬编码 SQL | `getFields()` 函数 | MySQL / PostgreSQL 各写一套 SQL，每次新增数据库就要加 `if-else` |
| 全局类型映射 | `dbType` 变量 | 所有数据库共用一份类型映射表，无法区分方言 |
| 紧耦合 | 整个 `gen.go` | 元数据获取、类型映射、模板渲染全揉在一个文件里 |

### 1.2 目标

支持任意数据库（MySQL、PostgreSQL、Manticore Search、SQLite、SQL Server 等），新增一个数据库只需：

1. 实现一个 **Dialect 接口**
2. 在 driver 包里注册即可

---

## 2. 核心设计：Dialect 接口

将数据库差异抽象为一个 `Dialect` 接口，每种数据库提供一个实现。

### 2.1 接口定义

```go
// ngen_new/dialect.go

package ngen

import "database/sql"

// Dialect 数据库方言接口，每种数据库实现自己的元数据获取和类型映射逻辑
type Dialect interface {
    // Name 返回方言名称，如 "mysql", "postgres", "manticore"
    Name() string

    // GetColumns 从数据库中获取表的所有列元数据
    // 返回 Field 列表和主键 Field 列表
    GetColumns(db *sql.DB, tableName string) (fields []Field, primaryKeys []Field, err error)

    // MapType 将数据库原生类型映射为 Go 类型信息
    // 返回: (Go类型, Null类型, Null取值后缀, 默认值)
    MapType(dbType string) (goType string, nullType string, nullValue string, defaultValue string)
}
```

### 2.2 为什么要拆成两个方法

- **`GetColumns`**: 每个数据库元数据查询方式完全不同
  - MySQL: `information_schema.COLUMNS`
  - PostgreSQL: `information_schema.columns` + `pg_catalog`
  - Manticore: `SHOW CREATE TABLE` / `DESCRIBE table` 解析（没有 information_schema）
  - SQLite: `PRAGMA table_info(table_name)`

- **`MapType`**: 每个数据库的原生类型名不同
  - MySQL 的 `VARCHAR` vs PostgreSQL 的 `varchar` / `character varying`
  - PostgreSQL 独有的 `UUID`, `JSONB`, `INET`
  - Manticore 的 `string`, `text`, `bigint`, `float`, `json`, `multi`, `mva64`

---

## 3. 文件拆分方案

```
ngen_new/
├── gen.go              # Generator 主逻辑 + Module/Field 数据结构（精简）
├── dialect.go          # Dialect 接口定义
├── dialect_mysql.go    # MySQL 实现
├── dialect_postgres.go # PostgreSQL 实现
├── dialect_manticore.go # Manticore Search 实现（新增）
├── template.go         # 所有模板函数（从 gen.go 底部抽离）
├── util.go             # 工具函数（驼峰、文件判断等）
└── gen_test.go         # 测试
```

### 3.1 各文件职责

| 文件 | 职责 | 预计行数 |
|------|------|----------|
| `gen.go` | `Generator` 结构体、`Module`/`Field` 定义、`Gen()` 主流程 | ~120 |
| `dialect.go` | `Dialect` 接口 + 方言注册表 + `GetDialect()` 工厂函数 | ~30 |
| `dialect_mysql.go` | MySQL 元数据 SQL + 类型映射 | ~120 |
| `dialect_postgres.go` | PostgreSQL 元数据 SQL + 类型映射 | ~120 |
| `dialect_manticore.go` | Manticore 元数据获取（SHOW/DESCRIBE） + 类型映射 | ~100 |
| `template.go` | 所有 `getXxxTemplate()` 函数 | ~250 |
| `util.go` | 字符串工具、文件工具 | ~60 |

---

## 4. 方言注册表机制

```go
// dialect.go

var dialects = map[string]Dialect{}

func RegisterDialect(d Dialect) {
    dialects[d.Name()] = d
}

func GetDialect(name string) (Dialect, error) {
    d, ok := dialects[name]
    if !ok {
        return nil, fmt.Errorf("unsupported driver: %s", name)
    }
    return d, nil
}
```

各实现在 `init()` 中自注册：

```go
// dialect_mysql.go
func init() {
    RegisterDialect(&MysqlDialect{})
}
```

---

## 5. Generator 重构

### 5.1 改造后的 Gen() 主流程

```go
func (gen *Generator) Gen(modules []Module) error {
    // 1. 校验参数
    // 2. 获取方言
    dialect, err := GetDialect(gen.driverName)
    if err != nil {
        return err
    }
    // 3. 连接数据库
    db, err := sql.Open(gen.driverName, gen.dsn)
    if err != nil {
        return err
    }
    // 4. 遍历 module，使用 dialect 获取字段信息
    for _, module := range modules {
        fields, primaryKeyFields, err := dialect.GetColumns(db, module.TableName)
        // ... 填充 module，调用 genFile()
    }
    return nil
}
```

### 5.2 Field 构建统一化

目前 `getFields` 里做了三件事：
1. 查询元数据
2. 类型映射
3. 拼装 Field 结构体（驼峰名、tag 等）

重构后第 1、2 步由 Dialect 负责，第 3 步抽成公共方法：

```go
// gen.go

// buildField 将列原始信息 + 类型映射 -> 完整的 Field 结构体
func buildField(columnName, columnType string, isNullable, isPrimaryKey int, comment string, columnDefault sql.NullString, dialect Dialect) Field {
    f := Field{
        ColumnName:       columnName,
        ColumnType:       columnType,
        IsNullable:       isNullable,
        IsPrimaryKey:     isPrimaryKey,
        Comment:          comment,
        ColumnDefault:    columnDefault,
        FieldName:        stringToUpperCamelCase(columnName),
        ColumnNameLowerCamel: stringToLowerCamelCase(columnName),
        ColumnNameUpper:  strings.ToUpper(columnName),
    }
    f.FieldType, f.FieldNullType, f.FieldNullTypeValue, f.FieldTypeDefault = dialect.MapType(columnType)
    f.FieldOrmTag  = fmt.Sprintf("orm:\"%v\"", columnName)
    f.FieldJsonTag = fmt.Sprintf("json:\"%v\"", columnName)
    f.FieldFormTag = fmt.Sprintf("form:\"%v\"", columnName)
    if val, _ := columnDefault.Value(); val != nil {
        f.FieldDefaultTag = fmt.Sprintf("default:\"%v\"", val)
    }
    return f
}
```

---

## 6. Manticore Search 方言实现要点

### 6.1 Manticore 特点

- 使用 MySQL 兼容协议（可用 `database/sql` + mysql driver 连接）
- **没有 `information_schema`**，需要用 `SHOW` 命令或 `DESCRIBE table` 获取元数据
- 字段类型有限：`string`、`text`、`bigint`、`integer`、`float`、`bool`、`json`、`mva64`、`multi`
- 没有传统的"主键"概念，`id` 字段是隐式的（`bigint` 类型，自动递增）
- 所有字段默认可为空，没有 `NOT NULL` 约束概念

### 6.2 获取表元数据

```sql
-- 方法 1：DESCRIBE（推荐，解析最稳定）
DESCRIBE table_name;

-- 返回: Field | Type | Properties
-- 示例: id | bigint | 
--       title | text | indexed
--       price | float | 

-- 方法 2：SHOW CREATE TABLE（需要解析 DDL）
SHOW CREATE TABLE table_name;
```

### 6.3 Manticore 类型映射

```go
var manticoreTypeMap = map[string]goType{
    "bigint":  {"int64", "sql.NullInt64", "Int64", "0"},
    "integer": {"int64", "sql.NullInt64", "Int64", "0"},
    "int":     {"int64", "sql.NullInt64", "Int64", "0"},
    "bool":    {"bool", "sql.NullBool", "Bool", "false"},
    "float":   {"float64", "sql.NullFloat64", "Float64", "0"},
    "string":  {"string", "sql.NullString", "String", "\"\""},
    "text":    {"string", "sql.NullString", "String", "\"\""},
    "json":    {"string", "sql.NullString", "String", "\"\""},
    "mva64":   {"string", "sql.NullString", "String", "\"\""},  // 多值属性，序列化为字符串
    "multi":   {"string", "sql.NullString", "String", "\"\""},
}
```

### 6.4 DESCRIBE 结果解析

Manticore 的 `DESCRIBE table_name` 返回三列：

| Field | Type | Properties |
|-------|------|------------|
| id | bigint | |
| title | text | indexed |
| price | float | |

解析时需要注意：
- **id 字段自动识别为主键**（Manticore 中 id 就是主键）
- Properties 列包含索引属性（`indexed`、`stored` 等），目前可忽略或存入 Comment
- 所有字段 `IsNullable` 默认 `false`

```go
func (d *ManticoreDialect) GetColumns(db *sql.DB, tableName string) ([]Field, []Field, error) {
    rows, err := db.Query("DESCRIBE " + tableName)
    if err != nil {
        return nil, nil, err
    }
    defer rows.Close()

    var fields, pkeys []Field
    for rows.Next() {
        var colName, colType, properties string
        if err := rows.Scan(&colName, &colType, &properties); err != nil {
            return nil, nil, err
        }
        colType = strings.ToLower(colType)
        isPK := 0
        if colName == "id" {
            isPK = 1
        }
        f := buildField(colName, colType, 0, isPK, properties, sql.NullString{}, d)
        if isPK == 1 {
            pkeys = append(pkeys, f)
        }
        fields = append(fields, f)
    }
    return fields, pkeys, nil
}
```

---

## 7. driver 包扩展

```go
// driver/driver.go 新增常量
const (
    DRIVER_NAME_POSTGRES  = "postgres"
    DRIVER_NAME_MYSQL     = "mysql"
    DRIVER_NAME_MANTICORE = "manticore"  // 新增
    PLACE_HOLDER_GO       = "ⒼⓄ"
)
```

注意：连接 Manticore 时 `sql.Open` 用的是 MySQL driver（因为 Manticore 兼容 MySQL 协议），`driverName` 传 `"mysql"`，但 `Dialect` 注册用 `"manticore"` 来区分，Generator 中需要两个概念分开。

建议在 `Generator` 中新增一个字段：

```go
type Generator struct {
    dsn         string
    project     string
    driverName  string  // sql.Open 用的 driver，如 "mysql", "postgres"
    dialectName string  // 方言选择，如 "mysql", "postgres", "manticore"
}
```

---

## 8. 迁移步骤

### Phase 1: 接口抽离（不改变行为）

1. 新建 `dialect.go`，定义 `Dialect` 接口 + 注册表
2. 新建 `dialect_mysql.go`，把原来 `getFields` 中 MySQL 逻辑 + `dbType` 搬进去
3. 新建 `dialect_postgres.go`，把 PostgreSQL 逻辑搬进去
4. 新建 `template.go`，把所有 `getXxxTemplate()` 搬出去
5. 修改 `gen.go` 的 `Gen()` 使用 `Dialect` 接口
6. 跑测试验证行为不变

### Phase 2: 新增 Manticore

7. 在 `driver/driver.go` 新增常量
8. 新建 `dialect_manticore.go`
9. 编写测试
10. 更新文档

### Phase 3: 后续可扩展

11. SQLite (`PRAGMA table_info`)
12. SQL Server (`sys.columns` + `sys.types`)

---

## 9. 文件变更总结

| 操作 | 文件 | 说明 |
|------|------|------|
| 新建 | `ngen_new/dialect.go` | Dialect 接口 + 注册表 |
| 新建 | `ngen_new/dialect_mysql.go` | MySQL 方言 |
| 新建 | `ngen_new/dialect_postgres.go` | PostgreSQL 方言 |
| 新建 | `ngen_new/dialect_manticore.go` | Manticore 方言 |
| 新建 | `ngen_new/template.go` | 模板函数集合 |
| 修改 | `ngen_new/gen.go` | 删除 getFields/dbType，改用 Dialect；删除模板函数 |
| 修改 | `driver/driver.go` | 新增 DRIVER_NAME_MANTICORE |
| 新建 | `docs/ngen_multi_database_design.md` | 本文档 |

---

## 10. 风险与注意事项

1. **Manticore 没有主键概念**：目前的 DAO 模板依赖主键来生成 `QuerySingleByPkey`、`Update`、`DeleteByPkey` 等方法。Manticore 中约定 `id` 字段就是主键，Dialect 在 `GetColumns` 里自动标记。

2. **类型映射不完整**：Manticore 的 `mva64`（多值整数属性）和 `multi`（多值字符串属性）在 Go 中没有直接对应类型，暂时映射为 `sql.NullString`，由业务层自行处理序列化。

3. **模板兼容性**：当前模板假设表一定有主键，Manticore 的表 ID 就是主键，可以兼容。如果未来支持无主键表（如日志表），需要调整模板生成逻辑。

4. **Manticore 不支持事务**：DAO 模板中 `Insert`/`Update` 等支持 `tx ...*sql.Tx` 参数，对 Manticore 虽然语法上兼容但实际不会生效。可以在 Dialect 接口中新增 `SupportsTransaction() bool` 方法，由模板按需判断。

5. **`driverName` vs `dialectName`**：Manticore 用 MySQL 协议，`sql.Open("mysql", dsn)` 连接，但方言是 `"manticore"`。Generator 需要区分这两个概念，建议新增 `DialectName()` 链式方法。