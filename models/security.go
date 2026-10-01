package models

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func RandomToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

const passwordPrefix = "$sublink$bcrypt-sha256$"

func hashPassword(password string) (string, error) {
	digest := sha256.Sum256([]byte(password))
	hash, err := bcrypt.GenerateFromPassword([]byte(hex.EncodeToString(digest[:])), bcrypt.DefaultCost)
	return passwordPrefix + string(hash), err
}

func verifyPasswordHash(hash, password string) error {
	if strings.HasPrefix(hash, passwordPrefix) {
		digest := sha256.Sum256([]byte(password))
		return bcrypt.CompareHashAndPassword([]byte(strings.TrimPrefix(hash, passwordPrefix)), []byte(hex.EncodeToString(digest[:])))
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func PrepareCredentials(user *User) error {
	if strings.TrimSpace(user.Username) == "" || len(user.Password) < 6 {
		return fmt.Errorf("账号不能为空，密码至少为 6 位")
	}
	password, err := hashPassword(user.Password)
	if err != nil {
		return err
	}
	user.Password = password
	user.SessionVersion, err = RandomToken()
	return err
}

// Upgrade existing databases without changing subscription contents or names.
func MigrateSecurity(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var users []User
		if err := tx.Find(&users).Error; err != nil {
			return err
		}
		for _, user := range users {
			updates := map[string]interface{}{}
			_, hashError := bcrypt.Cost([]byte(user.Password))
			if !strings.HasPrefix(user.Password, passwordPrefix) && hashError != nil {
				hash, err := hashPassword(user.Password)
				if err != nil {
					return err
				}
				updates["password"] = hash
			}
			if user.SessionVersion == "" {
				version, err := RandomToken()
				if err != nil {
					return err
				}
				updates["session_version"] = version
			}
			if len(updates) > 0 {
				if err := tx.Model(&User{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
					return err
				}
			}
		}
		var subscriptions []Subcription
		if err := tx.Where("token IS NULL OR token = ?", "").Find(&subscriptions).Error; err != nil {
			return err
		}
		for _, subscription := range subscriptions {
			token, err := RandomToken()
			if err != nil {
				return err
			}
			if err := tx.Model(&Subcription{}).Where("id = ?", subscription.ID).Update("token", token).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
