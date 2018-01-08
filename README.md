Go-Based regular-expression search. Based on ACK (https://beyondgrep.com/), created for fun, somewhat compatible.

# Installation
```
go get -u github.com/btm6084/goack
```

You can add your $GOPATH/bin to your $PATH to access it directly.
```
export PATH=$PATH:$GOPATH/bin
```

# Usage
```
goack [flags] <search term> [search directory]

eg.

goack -i "case (.+)[:]" .
```

# Implemented Flags

| Flag | Type | Description | Example
--- | --- | --- | ---
| `i` | Bool | Case insensitive search | -i
| `v` | Bool | Inverse Search. Returns all lines that *do not* match the search term | -v
| `l` | Bool | File Name Only | -l
| `follow` | Bool | Follow symlinks when building file search list. | -follow
| `A` | Int | Returns X lines AFTER the match | -A=5
| `B` | Int | Returns X lines BEFORE the match | -B=2
| `C` | Int | Returns X lines BEFORE and AFTER the match | -C=2
| `no-color` | Bool | Returns results with no color | --no-color

# goackrc configuration

Certain configuration options can be made permanent by adding a configuration file at /home/$USER/.goackrc/config.json

Currentl only ignore-dir is supported.

Example:
```
{
	"ignore-dir": [
		"vendor",
		"node_modules"
	]
}
```

# Vendoring
https://github.com/kardianos/govendor is the current vendoring solution of choice