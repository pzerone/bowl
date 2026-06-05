package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
)

func usage() {
	fmt.Fprint(os.Stderr, `bowl - a tiny container runtime

Usage:
  bowl run --rootfs <path> <command> [args...]

Flags:
  --rootfs   path to an already-extracted root filesystem (required)

Example:
  bowl run --rootfs ./alpine /bin/sh

Note: Bowl does not source roofts, you must provide one. Extract one first, e.g.
  mkdir alpine && docker export $(docker create alpine) | tar -x -C alpine
`)
}

func parseArgs(args []string) (rootfs string, cmdline []string, err error) {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.StringVar(&rootfs, "rootfs", "", "path to an extracted root filesystem")
	if err := fs.Parse(args); err != nil {
		return "", nil, err
	}
	if rootfs == "" {
		return "", nil, errors.New("--rootfs is required")
	}
	cmdline = fs.Args()
	if len(cmdline) == 0 {
		return "", nil, errors.New("no command specified")
	}
	return rootfs, cmdline, nil
}

// run re-executes bowl as a "child" inside fresh namespaces.
func run(args []string) error {
	rootfs, cmdline, err := parseArgs(args)
	if err != nil {
		return err
	}

	childArgs := append([]string{"child", "--rootfs", rootfs}, cmdline...)
	cmd := exec.Command("/proc/self/exe", childArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWUSER,
		Credential: &syscall.Credential{Uid: 0, Gid: 0},
		UidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getuid(), Size: 1},
		},
		GidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getgid(), Size: 1},
		},
	}
	return cmd.Run()
}

// Not meant to be invoked directly by users;
// run() re-execs bowl with this subcommand.
func child(args []string) error {
	rootfs, cmdline, err := parseArgs(args)
	if err != nil {
		return err
	}
	fmt.Printf("Running %v as pid %d\n", cmdline, os.Getpid())

	if err := syscall.Sethostname([]byte("container")); err != nil {
		return fmt.Errorf("sethostname: %w", err)
	}
	if err := syscall.Chroot(rootfs); err != nil {
		return fmt.Errorf("chroot %s: %w", rootfs, err)
	}
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("chdir /: %w", err)
	}
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("mount /proc: %w", err)
	}
	defer syscall.Unmount("/proc", 0)

	cmd := exec.Command(cmdline[0], cmdline[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "run":
		err = run(os.Args[2:])
	case "child":
		err = child(os.Args[2:])
	case "help", "-h", "--help":
		usage()
		return
	default:
		usage()
		os.Exit(1)
	}

	if err != nil {
		log.Fatal(err)
	}
}
