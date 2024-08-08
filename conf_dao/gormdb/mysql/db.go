package mysql

import (
	"fmt"
	pkdb "github.com/hopeio/initialize/conf_dao/gormdb"
	"github.com/hopeio/initialize/rootconf"
	dbi "github.com/hopeio/utils/dao/database"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Config pkdb.Config

func (c *Config) BeforeInjectWithRoot(conf *rootconf.RootConfig) {
	(*pkdb.Config)(c).BeforeInjectWithRoot(conf)
}

func (c *Config) Build() (*gorm.DB, error) {
	(*pkdb.Config)(c).Init()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%s&loc=%s",
		c.User, c.Password, c.Host,
		c.Port, c.Database, c.Charset, c.Mysql.ParseTime, c.Mysql.Loc)
	return (*pkdb.Config)(c).Build(mysql.Open(dsn))
}

type DB pkdb.DB

func (db *DB) Config() any {
	return (*Config)(&db.Conf)
}

func (db *DB) Init() error {
	var err error
	db.Conf.Type = dbi.Postgres
	db.DB, err = (*Config)(&db.Conf).Build()
	return err
}

func (db *DB) Close() error {
	return nil
}
