package driver

const (
	DRIVER_NAME_POSTGRES  = "postgres"
	DRIVER_NAME_PGX       = "pgx"
	DRIVER_NAME_MYSQL     = "mysql"
	DRIVER_NAME_MANTICORE = "manticore"
)

func IsPostgres(driverName string) bool {
	return driverName == DRIVER_NAME_POSTGRES || driverName == DRIVER_NAME_PGX
}

// IsPgx returns true if the driver is pgx native (not database/sql wrapper).
func IsPgx(driverName string) bool {
	return driverName == DRIVER_NAME_PGX
}
