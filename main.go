package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

const (
	InfoColor    = "\033[1;34m"
	NoticeColor  = "\033[1;36m"
	WarningColor = "\033[1;33m"
	ErrorColor   = "\033[1;31m"
	DebugColor   = "\033[0;36m"
	RESET        = "\033[0m"
)

func fatal(error string) {
	fmt.Printf(ErrorColor)
	log.Fatal(error, RESET)
}
func info(info string) {
	fmt.Printf(InfoColor)
	log.Println(info, RESET)
}
func debug(debug string) {
	fmt.Printf(DebugColor)
	log.Println(debug, RESET)
}
func success(success string) {
	fmt.Printf(NoticeColor)
	log.Println(success, RESET)
}
func fatalerr(err error) {
	if err != nil {
		fatal(err.Error())
	}
	success("done")
}
func removeEmptyStrings(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}
func find(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}
func chroot_exec(arg string) {
	val, err := exec.Command("su", "-c", "chroot "+arg).CombinedOutput()
	fmt.Println("su", "-c", "chroot "+arg)
	fmt.Println(string(val))
	fatalerr(err)
}
func chroot_install(arg, cache_dir, chroot_dir, old_pack string) {
	packs := removeEmptyStrings(strings.Split(string(arg), "\n"))
	for _, pack := range packs {
		dep_pack, err := exec.Command("su", "-c", "chroot "+chroot_dir+" sudo pacman -Sp "+pack+" --print-format %n").CombinedOutput()
		fatalerr(err)
		dep_packs := removeEmptyStrings(strings.Split(string(dep_pack), "\n"))
		if len(dep_packs) <= 1 || (len(dep_packs) == 2 && find(old_pack, dep_packs)) {
			debug("Installing " + pack)
			val, err := exec.Command("su", "-c", "chroot "+chroot_dir+" sudo pacman -Syq --overwrite='*' --noconfirm --needed "+pack).CombinedOutput()
			fmt.Println("su", "-c", "chroot", chroot_dir, "sudo pacman -Syq --overwrite='*' --noconfirm --needed", pack)
			fmt.Println(string(val))
			if !strings.Contains(string(val), "are in conflict") {
				fatalerr(err)
			}
			debug("Removing cache")
			_, err = exec.Command("/bin/sh", "-c", "rm -rf "+cache_dir).CombinedOutput()
			fatalerr(err)
		} else {
			for _, dep_pack := range dep_packs {
				chroot_install(dep_pack, cache_dir, chroot_dir, pack)
			}
		}
	}
}
func main() {
	fmt.Println("Linux for Phone")
	distro := flag.String("distro", "blackarch", "Write distribution name to install. arch, blackarch, debian and kali-linux are supported")
	arch := flag.String("arch", "aarch64", "write arch. Use only compatable one")
	//path := flag.String("repo_path", "http://mirror.archlinuxarm.org/", "Write repository path which should be used with distro. Defaults ['http://mirror.archlinuxarm.org/','http://ftp.debian.org/debian/','http://http.kali.org/kali/']")
	repo_url := flag.String("repo_url", "http://mirror.archlinuxarm.org/aarch64/core", "Path of core repository. ARCH ONLY.")
	//suite := flag.String("suite", "none", "Write realise which should be installed.KALI-LINUX AND DEBIAN ONLY. Defaults ['jessie','kali-rolling']")
	ipath := flag.String("image_path", "./", "Write path where image should be bulid")
	//igui := flag.Bool("gui", true, "Install gui on distro?")
	locale := flag.String("locale", "en_US.UTF-8", "Localization")
	flag.Parse()
	chroot_dir := *ipath + "linux"
	info("Installing Distro")
	debug("getting SU permission")
	_, err := exec.Command("su").CombinedOutput()
	if err != nil {
		fatal("Error: Please give root permission to start")
	}
	success("done")
	if *distro == "blackarch" || *distro == "arch" {
		debug("making chache directories")
		cache_dir := chroot_dir + "/var/cache/pacman/pkg"
		err = os.MkdirAll(cache_dir, 0777)
		fatalerr(err)
		debug("getting packages list")
		core_file, err := exec.Command("/bin/sh", "-c", "wget -q -O - '"+*repo_url+"/core.db.tar.gz' | tar xOz | grep '.pkg.tar.xz$' | grep -v -e '^linux-' -e '^grub-' -e '^efibootmgr-' -e '^openssh-' -e 'doc' -e 'amd-ucode' | sort").CombinedOutput()
		fatalerr(err)
		core_files := strings.Split(string(core_file), "\n")
		debug("getting packages")
		for _, pkg_file := range core_files {
			if pkg_file != "" {
				debug("Downloading " + pkg_file)
				_, err = exec.Command("wget", "-q", "-c", "-O", cache_dir+"/"+pkg_file, *repo_url+"/"+pkg_file).CombinedOutput()
				fatalerr(err)
				_, err = exec.Command("tar", "xJf", cache_dir+"/"+pkg_file, "-C", chroot_dir, "--exclude='./dev' --exclude='./sys' --exclude='./proc' --exclude='.INSTALL' --exclude='.MTREE' --exclude='.PKGINFO'").CombinedOutput()
				fatalerr(err)
			}
		}
		debug("configuring DNS")
		f, err := os.Create(chroot_dir + "/etc/resolv.conf")
		f.WriteString("nameserver 8.8.8.8")
		fatalerr(err)
		if _, err = os.Stat(chroot_dir + "/etc/nsswitch.conf"); err == nil {
			_, err = exec.Command("/bin/sh", "-c", "sed -i 's/systemd//g' "+chroot_dir+"/etc/nsswitch.conf").CombinedOutput()
			fatalerr(err)
		}
		debug("updating repo")
		_, err = exec.Command("sed", "-i", "s|^[[:space:]]*Architecture[[:space:]]*=.*$|Architecture = "+*arch+"|", chroot_dir+"/etc/pacman.conf").CombinedOutput()
		fatalerr(err)
		_, err = exec.Command("sed", "-i", `s|^[[:space:]]*\(CheckSpace\)|#\1|`, chroot_dir+"/etc/pacman.conf").CombinedOutput()
		fatalerr(err)
		_, err = exec.Command("sed", "-i", "s|^[[:space:]]*SigLevel[[:space:]]*=.*$|SigLevel = Never|", chroot_dir+"/etc/pacman.conf").CombinedOutput()
		fatalerr(err)
		f, err = os.OpenFile(chroot_dir+"/etc/pacman.d/mirrorlist", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		fatalerr(err)
		f.WriteString("Server = " + *repo_url)
		debug("Removing cache")
		output, err := exec.Command("/bin/sh", "-c", "rm -rf "+cache_dir).CombinedOutput()
		fatalerr(err)

		debug("Installing packages")
		output, err = exec.Command("/bin/sh", "-c", "echo '"+string(core_file)+"' | sed 's/-[0-9].*$//'").CombinedOutput()
		fatalerr(err)
		chroot_exec(chroot_dir + " sudo pacman -Sy")
		chroot_install(string(output), cache_dir, chroot_dir, "")
		debug("Removing cache")
		_, err = exec.Command("/bin/sh", "-c", "rm -rf "+cache_dir).CombinedOutput()
		fatalerr(err)
	}
	debug("configuring hostname")
	f, err := os.OpenFile(chroot_dir+"/etc/hostname", os.O_CREATE|os.O_WRONLY, 0644)
	fatalerr(err)
	f.WriteString("localhost")
	debug("configuring hosts")
	f, err = os.OpenFile(chroot_dir+"/etc/hosts", os.O_CREATE|os.O_WRONLY, 0644)
	fatalerr(err)
	f.WriteString("127.0.0.1 localhost")
	debug("configuring locale")
	f, err = os.OpenFile(chroot_dir+"/etc/locale.conf", os.O_CREATE|os.O_WRONLY, 0644)
	fatalerr(err)
	f.WriteString("LANG=" + *locale)
	debug("setting up su")
	for _, i := range []string{"/etc/pam.d/su", "/etc/pam.d/su-l"} {
		pam_su := chroot_dir + "/" + i
		_, err = exec.Command("sed", "-i", `1,/^auth/s/^\(auth.*\)$/auth\tsufficient\tpam_succeed_if.so uid = 0 use_uid quiet\n\1/`, pam_su).CombinedOutput()
		fatalerr(err)
	}
	debug("setting up sudo")
	sudo_str := "root ALL=(ALL:ALL)"
	output, err := exec.Command("sudo", "grep -q '"+sudo_str+"' "+chroot_dir+"/etc/sudoers").CombinedOutput()
	fatalerr(err)
	if string(output) == "" {
		chroot_exec(chroot_dir + " chmod 640 " + sudo_str + " " + chroot_dir + "/etc/sudoers")
		chroot_exec(chroot_dir + " /bin/sh -c echo " + sudo_str + " >> " + chroot_dir + "/etc/sudoers")
		chroot_exec(chroot_dir + " chmod 440 " + sudo_str + " " + chroot_dir + "/etc/sudoers")
	}
	success("Installation succeeded. Now copy directory to PC and put it in near folder with kernel")
}
