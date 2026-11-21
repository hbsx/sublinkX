package utils

import (
	"log"
	"os"
	"testing"
)

func TestLogInitDoesNotCreateFilesByDefaultOrWhenDisabled(t *testing.T) {
	directory, _ := os.Getwd()
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(directory)
	writer, flags := log.Writer(), log.Flags()
	defer func() { log.SetOutput(writer); log.SetFlags(flags) }()
	for _, mode := range []string{"", "console", "off"} {
		t.Setenv("SUBLINK_LOG_MODE", mode)
		Loginit()
		log.Println("non-sensitive test message")
		if _, err := os.Stat("logs"); !os.IsNotExist(err) {
			t.Fatal("unexpected disk log directory")
		}
	}
}
