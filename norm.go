package norm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-lazyer/norm/driver"
	"github.com/go-lazyer/norm/nsql"
	"github.com/go-lazyer/north/ntype"
)

type DataSource struct {
	db           *sql.DB
	tx           *sql.Tx
	pgxPool      *pgxpool.Pool
	pgxTx        pgx.Tx
	DriverName   string
	DaoFilePaths []string
}
type Config struct {
	MaxOpenConns int
	MaxIdleConns int
}

func Open(driverName string, dsn string, config Config) (DataSource, error) {
	if driver.IsPgx(driverName) {
		return openPgxPool(dsn, config)
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return DataSource{}, err
	}
	err = db.Ping()
	if err != nil {
		return DataSource{}, err
	}
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)

	return DataSource{
		db:         db,
		DriverName: driverName,
	}, nil
}

func openPgxPool(dsn string, config Config) (DataSource, error) {
	ctx := context.Background()
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return DataSource{}, err
	}
	poolConfig.MaxConns = int32(config.MaxOpenConns)
	poolConfig.MinConns = int32(config.MaxIdleConns)

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return DataSource{}, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return DataSource{}, err
	}

	return DataSource{
		pgxPool:    pool,
		DriverName: driver.DRIVER_NAME_PGX,
	}, nil
}

// SetTx sets a database/sql transaction on the DataSource for external use (e.g., generated DAO code).
func (ds *DataSource) SetTx(tx *sql.Tx) {
	ds.tx = tx
}

// SetPgxTx sets a pgx native transaction on the DataSource for external use.
func (ds *DataSource) SetPgxTx(tx pgx.Tx) {
	ds.pgxTx = tx
}

func CountByOrm(orm *nsql.CountOrm, ds DataSource) (int64, error) {
	orm.DriverName(ds.DriverName)
	sqlStr, params, err := orm.ToSql(true)
	if err != nil {
		return 0, err
	}
	return CountBySql(sqlStr, params, ds)
}

func CountBySql(sqlStr string, params []any, ds DataSource) (int64, error) {
	if sqlStr == "" || !strings.Contains(strings.ToLower(sqlStr), "count") {
		return 0, errors.New("must be a count sql")
	}
	if driver.IsPgx(ds.DriverName) {
		ctx := context.Background()
		var count int64
		if err := ds.pgxPool.QueryRow(ctx, sqlStr, params...).Scan(&count); err != nil {
			return 0, err
		}
		return count, nil
	}
	if ds.db == nil {
		return 0, errors.New("db not allowed to be nil,need to instantiate yourself")
	}
	row := ds.db.QueryRow(sqlStr, params...)
	var count int64
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// 查询
func QueryByOrm[T any](orm *nsql.SelectOrm, ds DataSource) ([]T, error) {
	if !driver.IsPgx(ds.DriverName) && ds.db == nil {
		return nil, errors.New("db not allowed to be nil,need to instantiate yourself")
	}
	if driver.IsPgx(ds.DriverName) && ds.pgxPool == nil {
		return nil, errors.New("pgx pool not allowed to be nil,need to instantiate yourself")
	}
	orm.DriverName(ds.DriverName)
	sqlStr, params, err := orm.ToSql(true)
	if err != nil {
		return nil, err
	}
	return QueryBySql[T](sqlStr, params, ds)
}
func QueryBySql[T any](sqlStr string, params []any, ds DataSource) ([]T, error) {
	if sqlStr == "" || !strings.Contains(strings.ToLower(sqlStr), "select") {
		return nil, errors.New("must be a select sql")
	}

	if driver.IsPgx(ds.DriverName) {
		ctx := context.Background()
		rows, err := ds.pgxPool.Query(ctx, sqlStr, params...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return pgx.CollectRows(rows, pgx.RowToStructByNameLax[T])
	}

	if ds.db == nil {
		return nil, errors.New("db not allowed to be nil")
	}

	rows, err := ds.db.Query(sqlStr, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return RowsToStruct[T](rows)
}

// InsertByMap 在 PostgreSQL 下单条插入，主键列需由调用方显式传入。
func InsertByMap(tableName, primaryKey string, insertMap map[string]any, ds DataSource) (int64, error) {
	if tableName == "" {
		return 0, errors.New("tableName is empty")
	}
	if strings.TrimSpace(primaryKey) == "" {
		return 0, errors.New("primaryKey is empty")
	}
	if len(insertMap) == 0 {
		return 0, errors.New("insertMap is empty")
	}
	if !driver.IsPgx(ds.DriverName) && ds.db == nil {
		return 0, errors.New("db not allowed to be nil,need to instantiate yourself")
	}
	orm := nsql.NewInsertOrm().DriverName(ds.DriverName).Table(tableName).PrimaryKey(primaryKey).Insert(insertMap)
	return InsertByOrm(orm, ds)
}

func InsertByOrm(orm *nsql.InsertOrm, ds DataSource) (int64, error) {
	if !driver.IsPgx(ds.DriverName) && ds.db == nil {
		return 0, errors.New("db not allowed to be nil,need to instantiate yourself")
	}
	orm.DriverName(ds.DriverName)
	sqlStr, params, err := orm.ToSql(true)
	if err != nil {
		return 0, err
	}
	return InsertBySql(sqlStr, params, ds)
}

// InsertBySql 执行单条插入，返回插入记录的 ID。
func InsertBySql(sqlStr string, params []any, ds DataSource) (int64, error) {

	if sqlStr == "" || !strings.Contains(strings.ToLower(sqlStr), "insert") {
		return 0, errors.New("must be a insert sql")
	}

	if driver.IsPgx(ds.DriverName) {
		if !strings.Contains(strings.ToUpper(sqlStr), "RETURNING") {
			return 0, errors.New("pgx insert must contain RETURNING clause")
		}
		ctx := context.Background()
		var row pgx.Row
		if ds.pgxTx != nil {
			row = ds.pgxTx.QueryRow(ctx, sqlStr, params...)
		} else {
			row = ds.pgxPool.QueryRow(ctx, sqlStr, params...)
		}
		var rawID any
		if err := row.Scan(&rawID); err != nil {
			return 0, err
		}
		return ntype.ToInt64(rawID), nil
	}

	if ds.db == nil && ds.tx == nil {
		return 0, errors.New("db and tx not allowed to be all nil,need to instantiate yourself")
	}

	var ret sql.Result
	var err error
	if ds.tx != nil {
		ret, err = ds.tx.Exec(sqlStr, params...) // 如果有事务，则在事务中执行
	} else if ds.db != nil {
		ret, err = ds.db.Exec(sqlStr, params...) // 直接执行插入语句
	}
	if err != nil {
		return 0, err
	}
	id, err := ret.LastInsertId() // 新插入数据的id
	if err != nil {
		return 0, err
	}
	return id, nil
}

// InsertsByMap 在 PostgreSQL 下批量插入，主键列需由调用方显式传入。
func InsertsByMap(tableName, primaryKey string, insertMap []map[string]any, ds DataSource) (int64, error) {
	if tableName == "" {
		return 0, errors.New("tableName is empty")
	}
	if strings.TrimSpace(primaryKey) == "" {
		return 0, errors.New("primaryKey is empty")
	}
	if len(insertMap) == 0 {
		return 0, errors.New("insertMap is empty")
	}
	if !driver.IsPgx(ds.DriverName) && ds.db == nil {
		return 0, errors.New("db not allowed to be nil,need to instantiate yourself")
	}
	orm := nsql.NewInsertOrm().DriverName(ds.DriverName).Table(tableName).PrimaryKey(primaryKey).Insert(insertMap...)
	return InsertsByOrm(orm, ds)
}

// InsertsByOrm 批量插入，返回成功插入的条数。
func InsertsByOrm(orm *nsql.InsertOrm, ds DataSource) (int64, error) {
	if !driver.IsPgx(ds.DriverName) && ds.db == nil {
		return 0, errors.New("db not allowed to be nil,need to instantiate yourself")
	}
	orm.DriverName(ds.DriverName)
	sqlStr, params, err := orm.ToSql(true)
	if err != nil {
		return 0, err
	}
	return InsertsBySql(sqlStr, params, ds)
}

// InsertsBySql 批量插入，返回成功插入的条数。
func InsertsBySql(sqlStr string, params []any, ds DataSource) (int64, error) {
	if sqlStr == "" || !strings.Contains(strings.ToLower(sqlStr), "insert") {
		return 0, errors.New("must be a insert sql")
	}
	if len(params) == 0 {
		return 0, nil
	}

	if driver.IsPgx(ds.DriverName) {
		ctx := context.Background()
		var tag pgconn.CommandTag
		var err error
		if ds.pgxTx != nil {
			tag, err = ds.pgxTx.Exec(ctx, sqlStr, params...)
		} else {
			tag, err = ds.pgxPool.Exec(ctx, sqlStr, params...)
		}
		if err != nil {
			return 0, err
		}
		return tag.RowsAffected(), nil
	}

	if ds.db == nil && ds.tx == nil {
		return 0, errors.New("db and tx not allowed to be all nil,need to instantiate yourself")
	}
	var ret sql.Result
	var err error
	if ds.tx != nil {
		ret, err = ds.tx.Exec(sqlStr, params...)
	} else {
		ret, err = ds.db.Exec(sqlStr, params...)
	}
	if err != nil {
		return 0, err
	}
	rowsAffected, err := ret.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rowsAffected, nil
}

func UpdateByOrm(orm *nsql.UpdateOrm, ds DataSource) (int64, error) {
	if !driver.IsPgx(ds.DriverName) && ds.db == nil {
		return 0, errors.New("db not allowed to be nil,need to instantiate yourself")
	}
	orm.DriverName(ds.DriverName)
	sqlStr, params, err := orm.ToSql(true)
	if err != nil {
		return 0, err
	}
	return UpdateBySql(sqlStr, params, ds)
}
func UpdateBySql(sqlStr string, params []any, ds DataSource) (int64, error) {

	if sqlStr == "" || !strings.Contains(strings.ToLower(sqlStr), "update") {
		return 0, errors.New("must be a update sql")
	}

	if driver.IsPgx(ds.DriverName) {
		ctx := context.Background()

		var tag pgconn.CommandTag
		var err error
		if ds.pgxTx != nil {
			tag, err = ds.pgxTx.Exec(ctx, sqlStr, params...)
		} else {
			tag, err = ds.pgxPool.Exec(ctx, sqlStr, params...)
		}
		if err != nil {
			return 0, err
		}
		return tag.RowsAffected(), nil
	}

	if ds.db == nil && ds.tx == nil {
		return 0, errors.New("db and tx not allowed to be all nil,need to instantiate yourself")
	}

	var ret sql.Result
	var err error
	if ds.tx != nil {
		ret, err = ds.tx.Exec(sqlStr, params...) // 如果有事务，则在事务中执行
	} else if ds.db != nil {
		ret, err = ds.db.Exec(sqlStr, params...) // 直接执行插入语句
	}
	if err != nil {
		return 0, err
	}
	num, err := ret.RowsAffected() // 操作影响的行数
	if err != nil {
		return 0, err
	}
	return num, nil
}

func UpdatesByOrm(orm *nsql.UpdateOrm, ds DataSource) ([]int64, error) {
	if !driver.IsPgx(ds.DriverName) && ds.db == nil {
		return nil, errors.New("db not allowed to be nil,need to instantiate yourself")
	}
	orm.DriverName(ds.DriverName)
	sqlStr, params, err := orm.ToSql(true)
	if err != nil {
		return nil, err
	}
	return UpdatesBySql(sqlStr, params, ds)
}

func UpdatesBySql(sqlStr string, params []any, ds DataSource) ([]int64, error) {
	if sqlStr == "" || !strings.Contains(strings.ToUpper(sqlStr), "CASE") {
		return nil, errors.New("must be a update use CASE WHEN sql")
	}
	n, err := UpdateBySql(sqlStr, params, ds)
	if err != nil {
		return nil, err
	}
	return []int64{n}, nil
}

func DeleteByOrm(orm *nsql.DeleteOrm, ds DataSource) (int64, error) {
	if !driver.IsPgx(ds.DriverName) && ds.db == nil {
		return 0, errors.New("db not allowed to be nil,need to instantiate yourself")
	}
	orm.DriverName(ds.DriverName)
	sqlStr, params, err := orm.ToSql(true)
	if err != nil {
		return 0, err
	}
	return DeleteBySql(sqlStr, params, ds)
}
func DeleteBySql(sqlStr string, params []any, ds DataSource) (int64, error) {
	if sqlStr == "" || !strings.Contains(strings.ToLower(sqlStr), "delete") {
		return 0, errors.New("must be a delete sql")
	}

	if driver.IsPgx(ds.DriverName) {
		ctx := context.Background()

		var tag pgconn.CommandTag
		var err error
		if ds.pgxTx != nil {
			tag, err = ds.pgxTx.Exec(ctx, sqlStr, params...)
		} else {
			tag, err = ds.pgxPool.Exec(ctx, sqlStr, params...)
		}
		if err != nil {
			return 0, err
		}
		return tag.RowsAffected(), nil
	}

	if ds.db == nil && ds.tx == nil {
		return 0, errors.New("db and tx not allowed to be all nil,need to instantiate yourself")
	}

	var ret sql.Result
	var err error
	if ds.tx != nil {
		ret, err = ds.tx.Exec(sqlStr, params...) // 如果有事务，则在事务中执行
	} else if ds.db != nil {
		ret, err = ds.db.Exec(sqlStr, params...) // 直接执行删除语句
	}
	if err != nil {
		return 0, err
	}
	num, err := ret.RowsAffected() // 操作影响的行数
	if err != nil {
		return 0, err
	}
	return num, nil
}

func UpsertByMap(tableName string, insertMap map[string]any, conflict []string, ds DataSource) (int64, error) {
	if tableName == "" {
		return 0, errors.New("tableName is empty")
	}
	if !driver.IsPgx(ds.DriverName) && ds.db == nil {
		return 0, errors.New("db not allowed to be nil,need to instantiate yourself")
	}
	if len(insertMap) == 0 {
		return 0, nil
	}
	orm := nsql.NewUpsertOrm().Table(tableName).Insert(insertMap).Update(nsql.MapKeys(insertMap)).Conflict(conflict...).DriverName(ds.DriverName)
	return UpsertByOrm(orm, ds)
}

func UpsertByOrm(orm *nsql.UpsertOrm, ds DataSource) (int64, error) {
	if !driver.IsPgx(ds.DriverName) && ds.db == nil {
		return 0, errors.New("db not allowed to be nil,need to instantiate yourself")
	}
	orm.DriverName(ds.DriverName)
	sqlStr, params, err := orm.ToSql(true)
	if err != nil {
		return 0, err
	}
	return UpsertBySql(sqlStr, params, ds)
}
func UpsertBySql(sqlStr string, params []any, ds DataSource) (int64, error) {

	if sqlStr == "" || !strings.Contains(strings.ToLower(sqlStr), "on") {
		return 0, errors.New("must be a upsert sql")
	}

	if driver.IsPgx(ds.DriverName) {
		ctx := context.Background()

		var tag pgconn.CommandTag
		var err error
		if ds.pgxTx != nil {
			tag, err = ds.pgxTx.Exec(ctx, sqlStr, params...)
		} else {
			tag, err = ds.pgxPool.Exec(ctx, sqlStr, params...)
		}
		if err != nil {
			return 0, fmt.Errorf("error%v;sql: %s", err, sqlStr)
		}
		return tag.RowsAffected(), nil
	}

	if ds.db == nil && ds.tx == nil {
		return 0, errors.New("db and tx not allowed to be all nil,need to instantiate yourself")
	}
	var ret sql.Result
	var err error
	if ds.tx != nil {
		ret, err = ds.tx.Exec(sqlStr, params...) // 如果有事务，则在事务中执行
	} else if ds.db != nil {
		ret, err = ds.db.Exec(sqlStr, params...) // 直接执行插入语句
	}
	if err != nil {
		return 0, fmt.Errorf("error%v;sql: %s", err, sqlStr)
	}
	num, err := ret.RowsAffected() // 操作影响的行数
	if err != nil {
		return 0, fmt.Errorf("error%v", err)
	}
	return num, nil
}

func UpsertsByMap(tableName string, insertMap []map[string]any, conflict []string, ds DataSource) ([]int64, error) {
	if tableName == "" {
		return []int64{}, errors.New("tableName is empty")
	}
	if !driver.IsPgx(ds.DriverName) && ds.db == nil {
		return []int64{}, errors.New("db not allowed to be nil,need to instantiate yourself")
	}
	if len(insertMap) == 0 {
		return []int64{}, nil
	}
	orm := nsql.NewUpsertOrm().Table(tableName).Insert(insertMap...).Update(nsql.MapKeys(insertMap[0])).Conflict(conflict...).DriverName(ds.DriverName)
	return UpsertsByOrm(orm, ds)
}

func UpsertsByOrm(orm *nsql.UpsertOrm, ds DataSource) ([]int64, error) {
	if !driver.IsPgx(ds.DriverName) && ds.db == nil {
		return []int64{}, errors.New("db not allowed to be nil,need to instantiate yourself")
	}
	orm.DriverName(ds.DriverName)
	sqlStr, params, err := orm.ToSql(true)
	if err != nil {
		return nil, err
	}
	return UpsertsBySql(sqlStr, params, ds)
}

func UpsertsBySql(sqlStr string, params []any, ds DataSource) ([]int64, error) {
	if sqlStr == "" || !strings.Contains(strings.ToUpper(sqlStr), "ON") {
		return nil, errors.New("must be a upsert sql with ON CONFLICT / ON DUPLICATE KEY")
	}
	n, err := UpsertBySql(sqlStr, params, ds)
	if err != nil {
		return nil, err
	}
	return []int64{n}, nil
}

func ExecBySql(sqlStr string, ds DataSource) (int64, error) {

	if sqlStr == "" {
		return 0, errors.New("sql must not be empty")
	}

	if driver.IsPgx(ds.DriverName) {
		ctx := context.Background()

		var tag pgconn.CommandTag
		var err error
		if ds.pgxTx != nil {
			tag, err = ds.pgxTx.Exec(ctx, sqlStr)
		} else {
			tag, err = ds.pgxPool.Exec(ctx, sqlStr)
		}
		if err != nil {
			return 0, err
		}
		return tag.RowsAffected(), nil
	}

	if ds.db == nil {
		return 0, errors.New("db not allowed to be nil,need to instantiate yourself")
	}

	ret, err := ds.db.Exec(sqlStr)
	if err != nil {
		return 0, err
	}
	return ret.RowsAffected()
}

// 把查询结果映射为实体
func RowsToStruct[T any](rows *sql.Rows) ([]T, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// 获取类型 T 的反射信息
	structType := reflect.TypeOf(new(T)).Elem()
	if structType.Kind() == reflect.Ptr {
		return nil, errors.New("t must be a non-pointer type")
	}

	// 创建值类型的切片（[]T）
	sliceType := reflect.SliceOf(structType)
	sliceValue := reflect.MakeSlice(sliceType, 0, 0)

	// 递归获取所有字段及其对应的 db 标签
	fieldToColIndex, err := getAllFieldToColIndex(structType, columns)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		elemPtr := reflect.New(structType)
		elemValue := elemPtr.Elem()

		scanArgs := make([]any, len(columns))
		for i := range scanArgs {
			var temp interface{}
			scanArgs[i] = &temp
		}

		for fieldName, index := range fieldToColIndex {
			field := elemValue.FieldByName(fieldName)
			if !field.IsValid() || !field.CanAddr() {
				return nil, fmt.Errorf("field %s not found or not addressable in type %T", fieldName, elemValue.Interface())
			}
			scanArgs[index] = field.Addr().Interface()
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, err
		}
		sliceValue = reflect.Append(sliceValue, elemValue)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sliceValue.Interface().([]T), nil
}

// 获取表字段和 struct 字段的并集
func getAllFieldToColIndex(structType reflect.Type, columns []string) (map[string]int, error) {
	fieldMap := forEachField(structType)
	fieldToColIndex := make(map[string]int)
	for i, columnName := range columns {
		if fieldName, ok := fieldMap[columnName]; ok {
			fieldToColIndex[fieldName] = i
		} else {
			fmt.Printf("table column %s not found in struct fields\n", columnName)
		}
	}
	return fieldToColIndex, nil
}

// 遍历结构体的所有字段和 tag 中的 db 标签，包括继承的 struct
func forEachField(structType reflect.Type) map[string]string {
	fields := make(map[string]string, 0)

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			// 如果是嵌套结构体，递归处理
			maps.Copy(fields, forEachField(field.Type))
		} else {
			tagValue := field.Tag.Get("db")
			if tagValue != "" {
				fields[field.Tag.Get("db")] = field.Name
			}
		}
	}
	return fields
}

func (ds DataSource) Transaction(fc func(tx *sql.Tx) error) (err error) {
	tx, err := ds.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	if err = fc(tx); err == nil {
		return tx.Commit()
	}
	return
}

// PgxTransaction executes fn within a pgx native transaction, committing on success or rolling back on error.
func (ds DataSource) PgxTransaction(ctx context.Context, fn func(tx pgx.Tx) error) (err error) {
	tx, err := ds.pgxPool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	if err = fn(tx); err == nil {
		return tx.Commit(ctx)
	}
	return
}
