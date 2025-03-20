package gluawatch

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	lua "github.com/yuin/gopher-lua"
)

// Loader is the module loader function for gluawatch
func Loader(L *lua.LState) int {
	// Create a module table
	mod := L.NewTable()
	L.SetFuncs(mod, map[string]lua.LGFunction{
		"watch": watch,
	})
	L.Push(mod)
	return 1
}

var ignoredPaths = []string{".git", "node_modules", "vendor", "__pycache__"}

type Debouncer struct {
	mu       sync.Mutex
	timers   map[string]*time.Timer
	delay    time.Duration
	luaState *lua.LState
	luaFunc  *lua.LFunction
}

func newDebouncer(L *lua.LState, luaFunc *lua.LFunction, delay time.Duration) *Debouncer {
	return &Debouncer{
		timers:   make(map[string]*time.Timer),
		delay:    delay,
		luaState: L,
		luaFunc:  luaFunc,
	}
}

func (d *Debouncer) trigger(file string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if timer, exists := d.timers[file]; exists {
		timer.Stop()
	}

	d.timers[file] = time.AfterFunc(d.delay, func() {
		d.mu.Lock()
		delete(d.timers, file)
		d.mu.Unlock()

		d.luaState.CallByParam(lua.P{
			Fn:      d.luaFunc,
			NRet:    0,
			Protect: true,
		}, lua.LString(file))
	})
}

func isIgnoredPath(path string) bool {
	for _, ignored := range ignoredPaths {
		if strings.Contains(path, ignored) {
			return true
		}
	}
	return false
}
func addRecursiveWatcher(watcher *fsnotify.Watcher, rootPath string) error {
	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && !isIgnoredPath(path) {
			err := watcher.Add(path)
			if err != nil {
				log.Printf("Error watching %s: %v", path, err)
			}
		}
		return nil
	})
}

func watch(L *lua.LState) int {
	// Get paths from Lua (expected as a table of strings)
	pathsTable := L.CheckTable(1)
	paths := []string{}
	pathsTable.ForEach(func(_, value lua.LValue) {
		if str, ok := value.(lua.LString); ok {
			paths = append(paths, string(str))
		}
	})

	// Get the Lua callback function
	luaFunc := L.CheckFunction(2)

	// Optional debounce delay (default 500ms)
	delay := time.Duration(L.OptInt(3, 500)) * time.Millisecond

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		L.Push(lua.LString(fmt.Sprintf("Error creating watcher: %v", err)))
		return 1
	}

	debouncer := newDebouncer(L, luaFunc, delay)

	// Add paths recursively
	for _, path := range paths {
		if err := addRecursiveWatcher(watcher, path); err != nil {
			L.Push(lua.LString(fmt.Sprintf("Error adding path %s: %v", path, err)))
			return 1
		}
	}

	// Goroutine to handle events
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if isIgnoredPath(event.Name) {
					continue
				}

				debouncer.trigger(event.Name)

				if event.Op&fsnotify.Create == fsnotify.Create {
					fileInfo, err := filepath.Glob(event.Name)
					if err == nil && fileInfo != nil {
						watcher.Add(event.Name)
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Watcher error:", err)
			}
		}
	}()

	L.Push(lua.LNil)
	return 1
}
