package models

import (
	"log"
	"os"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB
var isInitialized bool

func InitSqlite() {
	if err := os.MkdirAll("./db", 0755); err != nil {
		log.Fatal("创建数据库目录失败: ", err)
	}
	// 连接数据库
	db, err := gorm.Open(sqlite.Open("./db/sublink.db"), &gorm.Config{Logger: logger.New(
		log.New(os.Stderr, "", log.LstdFlags), logger.Config{
			SlowThreshold: time.Second, LogLevel: logger.Warn,
			IgnoreRecordNotFoundError: true, ParameterizedQueries: true,
		})})
	if err != nil {
		log.Fatal("连接数据库失败: ", err)
	}
	DB = db
	// 检查是否已经初始化
	if isInitialized {
		log.Println("数据库已经初始化，无需重复初始化")
		return
	}
	err = db.AutoMigrate(&User{}, &Subcription{}, &SubLogs{}, &GroupNode{}, &Node{})
	if err != nil {
		log.Fatal("数据表迁移失败: ", err)
	}
	if err := MigrateSecurity(db); err != nil {
		log.Fatal("安全配置迁移失败: ", err)
	}
	if err := MigrateSubscriptionOrder(db); err != nil {
		log.Fatal("订阅排序迁移失败: ", err)
	}
	// 初始化用户数据
	err = db.First(&User{}).Error
	if err == gorm.ErrRecordNotFound {
		admin := &User{
			Username: "admin",
			Password: "123456",
			Role:     "admin",
			Nickname: "管理员",
		}
		err = admin.Create()
		if err != nil {
			log.Fatal("初始化添加用户数据失败: ", err)
		}
	} else if err != nil {
		log.Fatal("读取用户数据失败: ", err)
	}
	// 设置初始化标志为 true
	isInitialized = true
	log.Println("数据库初始化成功") // 只有在没有任何错误时才会打印这个日志
}
