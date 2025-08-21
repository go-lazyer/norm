package nsql

type Orm interface {
	Table(table string) *Orm
}
