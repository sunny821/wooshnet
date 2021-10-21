package main

import (
	"flag"
	"os"
	"runtime"

	"wooshnet/pkg/cniapi"

	"github.com/containernetworking/cni/pkg/version"

	"github.com/containernetworking/cni/pkg/skel"
)

func init() {
	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()
	os.Setenv("TZ", "Asia/Shanghai")
}

func main() {
	flag.Parse()

	// log.Println(os.Args)
	// log.Println(os.Environ())
	skel.PluginMain(cniapi.CmdAdd, cniapi.CmdCheck, cniapi.CmdDel, version.All, "wooshcni")
}
