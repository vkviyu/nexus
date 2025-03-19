package gormdb

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func Open(mysqlConfig mysql.Config, gormConfig *gorm.Config) (*gorm.DB, error) {
	if gormConfig == nil {
		gormConfig = &gorm.Config{}
	}
	db, err := gorm.Open(mysql.New(mysqlConfig), gormConfig)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func OpenWithDSN(dsn string, gormConfig *gorm.Config) (*gorm.DB, error) {
	if gormConfig == nil {
		gormConfig = &gorm.Config{}
	}
	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func Get[T any](db *gorm.DB, query interface{}, args ...interface{}) (*gorm.DB, []T) {
	var dest []T
	result := db.Where(query, args...).Find(&dest)
	return result, dest
}

func GetWithDeleted[T any](db *gorm.DB, query interface{}, args ...interface{}) (*gorm.DB, []T) {
	var dest []T
	result := db.Unscoped().Where(query, args...).Find(&dest)
	return result, dest
}

func FindOne[T any](db *gorm.DB, query interface{}, args ...interface{}) (*gorm.DB, *T) {
	dest := new(T)
	result := db.Where(query, args...).Limit(1).Find(dest)
	if result.RowsAffected == 0 {
		return result, nil
	}
	return result, dest
}

func FindOneWithDeleted[T any](db *gorm.DB, query interface{}, args ...interface{}) (*gorm.DB, *T) {
	dest := new(T)
	result := db.Unscoped().Where(query, args...).Limit(1).Find(dest)
	if result.RowsAffected == 0 {
		return result, nil
	}
	return result, dest
}

func FindOneOrErrNotFound[T any](db *gorm.DB, query interface{}, args ...interface{}) (*gorm.DB, *T) {
	dest := new(T) // allocate memory for a T instance
	result := db.Where(query, args...).First(dest)
	if result.RowsAffected == 0 {
		return result, nil
	}
	return result, dest
}

func FindOneWithDeletedOrErrNotFound[T any](db *gorm.DB, query interface{}, args ...interface{}) (*gorm.DB, *T) {
	dest := new(T) // allocate memory for a T instance
	result := db.Unscoped().Where(query, args...).First(dest)
	if result.RowsAffected == 0 {
		return result, nil
	}
	return result, dest
}
