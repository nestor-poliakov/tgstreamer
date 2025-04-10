package pg

type Config struct {
	Dsn        string `yaml:"dsn"`
	DropTables bool   `yaml:"drop_tables"`
}
