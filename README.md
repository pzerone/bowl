# Bowl

Bowl is a container runtime (like docker and podman), but very tiny.

It isolates a process using Linux namespaces (UTS, PID, mount, user) and
`chroot`s it into a root filesystem you provide. Bowl is rootless: container
UID/GID 0 is mapped to your real (unprivileged) host UID/GID.

## Sourcing a rootfs

Bowl does **not** pull or extract images — that's your job. Prepare an
extracted root filesystem first. For example, from a Docker image:

```sh
mkdir alpine
docker export "$(docker create alpine)" | tar -x -C alpine
```

## Usage

```
bowl run --rootfs <path> <command> [args...]
```

Example:

```sh
go build -o bowl .
./bowl run --rootfs ./alpine /bin/sh
```

The rootfs must contain a `/proc` directory (Bowl mounts procfs there).
