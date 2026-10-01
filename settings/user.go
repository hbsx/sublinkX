package settings

import (
	"fmt"
	"log"
	"sublink/models"

	"gorm.io/gorm"
)

// 重置默认用户
func ResetUser(username string, password string) error {
	// 如果账号或者密码为空
	if username == "" || password == "" {
		return fmt.Errorf("账号或者密码不能为空")
	}
	if len(password) < 6 {
		return fmt.Errorf("密码不能小于6位数")
	}
	err := models.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("1 = 1").Delete(&models.User{}).Error; err != nil {
			return err
		}
		user := &models.User{Username: username, Password: password, Role: "admin", Nickname: "管理员"}
		return tx.Create(user).Error
	})
	if err != nil {
		return fmt.Errorf("重置账号失败: %w", err)
	}
	log.Println("管理员账号已重置")
	return nil
}
