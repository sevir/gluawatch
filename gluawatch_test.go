package gluawatch

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	lua "github.com/yuin/gopher-lua"
)

func TestWatch(t *testing.T) {
	// Create temporary directory
	tmpDir, err := ioutil.TempDir("", "gluawatch")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := ioutil.WriteFile(testFile, []byte("initial content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	L := lua.NewState()
	defer L.Close()

	// Load the watch module
	L.PreloadModule("watch", Loader)

	// Channel to receive notifications from Lua callback
	notifications := make(chan string, 1)

	// Register callback function
	L.SetGlobal("notifyTest", L.NewFunction(func(L *lua.LState) int {
		fileName := L.ToString(1)
		notifications <- fileName
		return 0
	}))

	// Run the watch function
	if err := L.DoString(`
		local watch = require("watch")
		local paths = {"/tmp"}
		watch.watch(paths, notifyTest, 100)
	`); err != nil {
		t.Fatal(err)
	}

	// Wait a bit for the watcher to initialize
	time.Sleep(100 * time.Millisecond)

	// Modify the test file
	if err := ioutil.WriteFile(testFile, []byte("modified content"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	// Wait for notification
	select {
	case fileName := <-notifications:
		if fileName != testFile {
			t.Errorf("expected notification for %s, got %s", testFile, fileName)
		}
	case <-time.After(5 * time.Second):
		t.Error("timeout waiting for file change notification")
	}
}

func TestWatchInvalidPath(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("watch", Loader)

	// Test with non-existent path
	err := L.DoString(`
		local watch = require("watch")
		local paths = {"/nonexistent/path"}
		local err = watch.watch(paths, function() end)
		assert(err ~= nil, "Expected error for non-existent path")
	`)

	if err != nil {
		t.Error(err)
	}
}
