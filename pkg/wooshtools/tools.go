package wooshtools

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

func FuncName() string {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return "?"
	}

	fn := runtime.FuncForPC(pc)
	index := strings.LastIndexByte(fn.Name(), '.')
	if index == -1 {
		return fn.Name()
	}
	return fn.Name()[index+1:]
}

func FileLine() string {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		return "?:?"
	}
	return fmt.Sprintf("%s:%d", file, line)
}

func FileExist(path string) bool {
	_, err := os.Lstat(path)
	return !os.IsNotExist(err)
}
