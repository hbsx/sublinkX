package settings

import (
	"errors"
	"testing"

	"sublink/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestResetUserRollsBackOnCreateFailure(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	previousDB := models.DB
	models.DB = db
	t.Cleanup(func() { models.DB = previousDB })
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatal(err)
	}
	existing := models.User{Username: "existing", Password: "old-password"}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Create().Before("gorm:create").Register("test:fail_create", func(tx *gorm.DB) {
		tx.AddError(errors.New("simulated database write failure"))
	}); err != nil {
		t.Fatal(err)
	}
	if err := ResetUser("replacement", "new-password"); err == nil {
		t.Fatal("reset reported success despite a database write failure")
	}
	var retained models.User
	if err := db.First(&retained).Error; err != nil || retained.Username != "existing" {
		t.Fatal("failed reset removed the existing login")
	}
}
