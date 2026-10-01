package models

import (
	"fmt"
)

type User struct {
	ID             int
	Username       string
	Password       string `json:"-"`
	SessionVersion string `json:"-"`
	Role           string
	Nickname       string
}

func (user *User) Create() error { // 创建用户
	if err := PrepareCredentials(user); err != nil {
		return err
	}
	return DB.Create(user).Error
}
func (user *User) Set(UpdateUser *User) error { // 设置用户
	if err := PrepareCredentials(UpdateUser); err != nil {
		return err
	}
	result := DB.Model(&User{}).Where("username = ?", user.Username).Updates(map[string]interface{}{
		"username": UpdateUser.Username, "password": UpdateUser.Password,
		"session_version": UpdateUser.SessionVersion,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("用户不存在")
	}
	return nil
}
func (user *User) Verify() error { // 验证用户
	password := user.Password
	if err := user.Find(); err != nil {
		return err
	}
	return verifyPasswordHash(user.Password, password)
}

func (user *User) Find() error { // 查找用户
	return DB.Where("username = ? ", user.Username).First(user).Error
}

func (user *User) All() ([]User, error) { // 获取所有用户
	var users []User
	err := DB.Find(&users).Error
	return users, err
}

func (user *User) Del() error { // 删除用户
	return DB.Delete(user).Error
}
