package main

import (
	"io/ioutil"
	"log"
	"os"
	"sync/atomic"
	"syscall"

	"github.com/criyle/go-judge/pkg/pool"
	"github.com/criyle/go-sandbox/container"
	"github.com/criyle/go-sandbox/pkg/cgroup"
	"github.com/criyle/go-sandbox/pkg/forkexec"
)

func init() {
	container.Init()
}

func initEnvPool() {
	root, err := ioutil.TempDir("", "executorserver")
	if err != nil {
		log.Fatalln(err)
	}
	printLog("Created tmp dir for container root at:", root)

	mb, err := parseMountConfig(*mountConf)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatalln(err)
		}
		printLog("Use the default container mount")
		mb = getDefaultMount()
	}
	m, err := mb.Build(true)
	if err != nil {
		log.Fatalln(err)
	}
	printLog("Created container mount at:", mb)

	unshareFlags := uintptr(forkexec.UnshareFlags)
	if *netShare {
		unshareFlags ^= syscall.CLONE_NEWNET
	}

	b := &container.Builder{
		Root:          root,
		Mounts:        m,
		CredGenerator: newCredGen(),
		Stderr:        true,
		CloneFlags:    unshareFlags,
		ExecFile:      *cinitPath,
	}
	cgb, err := cgroup.NewBuilder("executorserver").WithCPUAcct().WithMemory().WithPids().FilterByEnv()
	if err != nil {
		log.Fatalln(err)
	}
	printLog("Created cgroup builder with:", cgb)

	cgroupPool := pool.NewFakeCgroupPool(cgb)
	builder := pool.NewEnvBuilder(b, cgroupPool)
	envPool = pool.NewPool(builder)
}

type credGen struct {
	cur uint32
}

func newCredGen() *credGen {
	return &credGen{cur: 10000}
}

func (c *credGen) Get() syscall.Credential {
	n := atomic.AddUint32(&c.cur, 1)
	return syscall.Credential{
		Uid: n,
		Gid: n,
	}
}
