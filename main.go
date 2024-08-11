package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/google/uuid"
)

func run() error {
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID,
	}
	return cmd.Run()
}
func child(dir string) error {
	fmt.Printf("Running: %v as %v\n", os.Args[2:], os.Getpid())
	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	syscall.Sethostname([]byte("container"))
	syscall.Chroot(dir)
	syscall.Chdir("/")
	syscall.Mount("proc", "proc", "proc", 0, "")
	return cmd.Run()
}

func setup() (string, error) {
	fmt.Println("Setting up dir for chroot")
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" { // XDG_DATA_HOME may not be set on some hacky distros
		homeDir, _ := os.UserHomeDir()
		dataHome = homeDir + "/.local/share"
	}
  newDir := fmt.Sprintf("%v/bowl/containers/%v", dataHome, uuid.New())
	if err := os.MkdirAll(newDir, 0775); err != nil {
		return "", err
	}
	return newDir, nil
}

func main() {
  chrootDir, err := setup()
  if err != nil {
    panic(err)
  }
	switch os.Args[1] {
	case "run":
		if err := run(); err != nil {
			panic(err)
		}
	case "child":
		if err := child(chrootDir); err != nil {
			panic(err)
		}
	default:
		panic(errors.New("Invalid cmdline arg"))
	}
}
