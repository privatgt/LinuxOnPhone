# LinuxOnPhone
building linux chroot with go (aarch64 and archlinux only)
## To build
to build just run
```go build main.go```
## Arguments (all optional)
repo_url - allow to add custom repo to install packages. Default http://mirror.archlinuxarm.org/aarch64/core\
image_path - allow to choose plase to make chroot directory. Default ./\
locale - allow to choose localization. Default en_US.UTF-8
## Note
this is just part of bigger project that will allow to port linux distro to phone without chroot
