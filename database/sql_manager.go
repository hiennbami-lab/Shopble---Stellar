package database

import (
	"time"

	"shopble/common/comrunner"
	"shopble/common/comtypes"
	"shopble/common/comutils"
	"shopble/config"

	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	vDbConnection = comtypes.NewSingleton(func() (dbContainer *SqlDb) {
		dsn := viper.GetString(config.KeyDbConnection)
		dbContainer, err := initDbFromDns(dsn)
		comutils.PanicOnError(err)
		return
	})
)

func InitDb() {
	GetDb()
}

func initDbFromDns(dns string) (*SqlDb, error) {
	db, err := gorm.Open(postgres.Open(dns), &gorm.Config{
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	comrunner.RegisterRootCloser(func() {
		_ = sqlDB.Close()
	})

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if config.Debug {
		db = db.Debug()
	}
	return &SqlDb{DB: db}, nil
}

func GetDb() (_ *SqlDb) {
	return vDbConnection.GetF()
}
