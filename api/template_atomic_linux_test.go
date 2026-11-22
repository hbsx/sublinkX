//go:build linux

package api

import (
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/gin-gonic/gin"
)

// A real short-write reproduces full-disk behavior without filling the host disk.
// Limit only an isolated child process so the rest of the test suite is untouched.
func TestTemplateAtomicSaveOnWriteFailure(t *testing.T) {
	if os.Getenv("SUBLINK_TEST_TEMPLATE_WRITE_LIMIT") != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestTemplateAtomicSaveOnWriteFailure$", "-test.v")
		cmd.Env = append(os.Environ(), "SUBLINK_TEST_TEMPLATE_WRITE_LIMIT=1")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("write-failure regression: %v\n%s", err, output)
		}
		return
	}
	baseTemplateDir = t.TempDir()
	file := filepath.Join(baseTemplateDir, "existing.yaml")
	original := "original-template"
	if err := os.WriteFile(file, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	signal.Ignore(syscall.SIGXFSZ)
	if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &syscall.Rlimit{Cur: 32, Max: 32}); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("POST", "/update", strings.NewReader(url.Values{"filename": {"existing.yaml"}, "oldname": {"existing.yaml"}, "text": {strings.Repeat("x", 128)}}.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = request
	UpdateTemp(c)
	if w.Code < 400 {
		t.Fatal("write unexpectedly succeeded")
	}
	data, err := os.ReadFile(file)
	if err != nil || string(data) != original {
		t.Fatal("failed save damaged the original template")
	}
	entries, err := os.ReadDir(baseTemplateDir)
	if err != nil || len(entries) != 1 {
		t.Fatal("temporary file was retained")
	}
}
