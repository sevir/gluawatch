# gluawatch

This is a lua module for watch file changes

## Usage

### Installation

```lua
local watch = require("gluawatch")
```

### Example

```lua
-- Watch one or more directories for changes
local paths = {"/path/to/watch", "/another/path"}
local delay = 500 -- debounce delay in milliseconds (optional, defaults to 500)

-- Callback function receives the changed file path
local function onChange(filepath)
    print("File changed:", filepath)
end

-- Start watching
watch.watch(paths, onChange, delay)
```

### API Reference

#### watch.watch(paths, callback[, delay])

- `paths`: table of strings - Directories to watch recursively
- `callback`: function(filepath) - Called when files change
- `delay`: number (optional) - Debounce delay in milliseconds, default 500

### Notes

The watcher automatically ignores these directories:
- .git
- node_modules
- vendor
- __pycache__
