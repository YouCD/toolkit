package db

import (
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 定义全局的db对象，我们执行数据库操作主要通过他实现。
var (
	db *gorm.DB
)

// InitDB
//
//	@Description: 初始化函数，golang特性，每个包初始化的时候会自动执行init函数，这里用来初始化gorm。
//	@param DBUser
//	@param DBPwd
//	@param DBHost
//	@param DBPort
//	@param DBName
func InitDB(user, pwd, host, port, name string, l logger.Interface) {
	DSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, pwd, host, port, name) // 连接数据库
	if pwd == "" {
		DSN = fmt.Sprintf("%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, host, port, name) // 连接数据库
	}

	if l != nil {
		l = logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
			logger.Config{
				SlowThreshold: time.Second,   // 慢 SQL 阈值
				LogLevel:      logger.Silent, // Log level
				Colorful:      false,         // 禁用彩色打印
			},
		)
	}

	var err error
	// 连接MYSQL, 获得DB类型实例，用于后面的数据库读写操作。
	db, err = gorm.Open(mysql.Open(DSN), &gorm.Config{
		Logger: l,
	})

	if err != nil {
		log.Println("连接数据库失败, error=" + err.Error())
		os.Exit(1)
	}
	// 设置数据库连接池参数

	sqlDB, err := db.DB()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	sqlDB.SetMaxOpenConns(100) // 设置数据库连接池最大连接数
	sqlDB.SetMaxIdleConns(20)  // 连接池最大允许的空闲连接数，如果没有sql任务需要执行的连接数大于20，超过的连接会被连接池关闭。
	// 设置每个链接的过期时间
	sqlDB.SetConnMaxLifetime(time.Second * 5)
	err = sqlDB.Ping()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

// GetDB 不用担心协程并发使用同样的db对象会共用同一个连接，db对象在调用他的方法的时候会从数据库连接池中获取新的连接
func GetDB() *gorm.DB {
	return db
}
