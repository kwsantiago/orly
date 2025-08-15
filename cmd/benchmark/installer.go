package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type DependencyType int

const (
	Go DependencyType = iota
	Rust
	Cpp
	Git
	Make
	Cmake
	Pkg
)

type RelayInstaller struct {
	workDir    string
	installDir string
	deps       map[DependencyType]bool
	mu         sync.RWMutex
	skipVerify bool
}

func NewRelayInstaller(workDir, installDir string) *RelayInstaller {
	return &RelayInstaller{
		workDir:    workDir,
		installDir: installDir,
		deps:       make(map[DependencyType]bool),
	}
}

func (ri *RelayInstaller) DetectDependencies() error {
	deps := []struct {
		dep DependencyType
		cmd string
	}{
		{Go, "go"},
		{Rust, "rustc"},
		{Cpp, "g++"},
		{Git, "git"},
		{Make, "make"},
		{Cmake, "cmake"},
		{Pkg, "pkg-config"},
	}

	ri.mu.Lock()
	defer ri.mu.Unlock()

	for _, d := range deps {
		_, err := exec.LookPath(d.cmd)
		ri.deps[d.dep] = err == nil
	}

	return nil
}

func (ri *RelayInstaller) InstallMissingDependencies() error {
	ri.mu.RLock()
	missing := make([]DependencyType, 0)
	for dep, exists := range ri.deps {
		if !exists {
			missing = append(missing, dep)
		}
	}
	ri.mu.RUnlock()

	if len(missing) == 0 {
		return nil
	}

	switch runtime.GOOS {
	case "linux":
		return ri.installLinuxDeps(missing)
	case "darwin":
		return ri.installMacDeps(missing)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func (ri *RelayInstaller) installLinuxDeps(deps []DependencyType) error {
	hasApt := ri.commandExists("apt-get")
	hasYum := ri.commandExists("yum")
	hasPacman := ri.commandExists("pacman")

	if !hasApt && !hasYum && !hasPacman {
		return fmt.Errorf("no supported package manager found")
	}

	if hasApt {
		if err := ri.runCommand("sudo", "apt-get", "update"); err != nil {
			return err
		}
	}

	for _, dep := range deps {
		switch dep {
		case Go:
			if err := ri.installGo(); err != nil {
				return err
			}
		case Rust:
			if err := ri.installRust(); err != nil {
				return err
			}
		default:
			if hasApt {
				if err := ri.installAptPackage(dep); err != nil {
					return err
				}
			} else if hasYum {
				if err := ri.installYumPackage(dep); err != nil {
					return err
				}
			} else if hasPacman {
				if err := ri.installPacmanPackage(dep); err != nil {
					return err
				}
			}
		}
	}

	if err := ri.installSecp256k1(); err != nil {
		return err
	}

	return nil
}

func (ri *RelayInstaller) installMacDeps(deps []DependencyType) error {
	if !ri.commandExists("brew") {
		return fmt.Errorf("homebrew not found, install from https://brew.sh")
	}

	for _, dep := range deps {
		switch dep {
		case Go:
			if err := ri.runCommand("brew", "install", "go"); err != nil {
				return err
			}
		case Rust:
			if err := ri.installRust(); err != nil {
				return err
			}
		case Cpp:
			if err := ri.runCommand("brew", "install", "gcc"); err != nil {
				return err
			}
		case Git:
			if err := ri.runCommand("brew", "install", "git"); err != nil {
				return err
			}
		case Make:
			if err := ri.runCommand("brew", "install", "make"); err != nil {
				return err
			}
		case Cmake:
			if err := ri.runCommand("brew", "install", "cmake"); err != nil {
				return err
			}
		case Pkg:
			if err := ri.runCommand("brew", "install", "pkg-config"); err != nil {
				return err
			}
		}
	}

	if err := ri.installSecp256k1(); err != nil {
		return err
	}

	return nil
}

func (ri *RelayInstaller) installAptPackage(dep DependencyType) error {
	var pkgName string
	switch dep {
	case Cpp:
		pkgName = "build-essential"
	case Git:
		pkgName = "git"
	case Make:
		pkgName = "make"
	case Cmake:
		pkgName = "cmake"
	case Pkg:
		pkgName = "pkg-config"
	default:
		return nil
	}

	return ri.runCommand("sudo", "apt-get", "install", "-y", pkgName, "autotools-dev", "autoconf", "libtool")
}

func (ri *RelayInstaller) installYumPackage(dep DependencyType) error {
	var pkgName string
	switch dep {
	case Cpp:
		pkgName = "gcc-c++"
	case Git:
		pkgName = "git"
	case Make:
		pkgName = "make"
	case Cmake:
		pkgName = "cmake"
	case Pkg:
		pkgName = "pkgconfig"
	default:
		return nil
	}

	return ri.runCommand("sudo", "yum", "install", "-y", pkgName)
}

func (ri *RelayInstaller) installPacmanPackage(dep DependencyType) error {
	var pkgName string
	switch dep {
	case Cpp:
		pkgName = "gcc"
	case Git:
		pkgName = "git"
	case Make:
		pkgName = "make"
	case Cmake:
		pkgName = "cmake"
	case Pkg:
		pkgName = "pkgconf"
	default:
		return nil
	}

	return ri.runCommand("sudo", "pacman", "-S", "--noconfirm", pkgName)
}

func (ri *RelayInstaller) installGo() error {
	version := "1.21.5"
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "amd64"
	} else if arch == "arm64" {
		arch = "arm64"
	}

	filename := fmt.Sprintf("go%s.%s-%s.tar.gz", version, runtime.GOOS, arch)
	url := fmt.Sprintf("https://golang.org/dl/%s", filename)

	tmpFile := filepath.Join(os.TempDir(), filename)
	if err := ri.runCommand("wget", "-O", tmpFile, url); err != nil {
		return fmt.Errorf("failed to download Go: %w", err)
	}

	if err := ri.runCommand("sudo", "tar", "-C", "/usr/local", "-xzf", tmpFile); err != nil {
		return fmt.Errorf("failed to extract Go: %w", err)
	}

	os.Remove(tmpFile)

	profile := filepath.Join(os.Getenv("HOME"), ".profile")
	f, err := os.OpenFile(profile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		f.WriteString("\nexport PATH=$PATH:/usr/local/go/bin\n")
		f.Close()
	}

	return nil
}

func (ri *RelayInstaller) installRust() error {
	return ri.runCommand("curl", "--proto", "=https", "--tlsv1.2", "-sSf", "https://sh.rustup.rs", "|", "sh", "-s", "--", "-y")
}

func (ri *RelayInstaller) installSecp256k1() error {
	switch runtime.GOOS {
	case "linux":
		if ri.commandExists("apt-get") {
			if err := ri.runCommand("sudo", "apt-get", "install", "-y", "libsecp256k1-dev"); err != nil {
				return ri.buildSecp256k1FromSource()
			}
			return nil
		} else if ri.commandExists("yum") {
			if err := ri.runCommand("sudo", "yum", "install", "-y", "libsecp256k1-devel"); err != nil {
				return ri.buildSecp256k1FromSource()
			}
			return nil
		} else if ri.commandExists("pacman") {
			if err := ri.runCommand("sudo", "pacman", "-S", "--noconfirm", "libsecp256k1"); err != nil {
				return ri.buildSecp256k1FromSource()
			}
			return nil
		}
		return ri.buildSecp256k1FromSource()
	case "darwin":
		if err := ri.runCommand("brew", "install", "libsecp256k1"); err != nil {
			return ri.buildSecp256k1FromSource()
		}
		return nil
	default:
		return ri.buildSecp256k1FromSource()
	}
}

func (ri *RelayInstaller) buildSecp256k1FromSource() error {
	secp256k1Dir := filepath.Join(ri.workDir, "secp256k1")

	if err := ri.runCommand("git", "clone", "https://github.com/bitcoin-core/secp256k1.git", secp256k1Dir); err != nil {
		return fmt.Errorf("failed to clone secp256k1: %w", err)
	}

	if err := os.Chdir(secp256k1Dir); err != nil {
		return err
	}

	if err := ri.runCommand("./autogen.sh"); err != nil {
		return fmt.Errorf("failed to run autogen: %w", err)
	}

	configArgs := []string{"--enable-module-schnorrsig", "--enable-module-recovery"}
	if err := ri.runCommand("./configure", configArgs...); err != nil {
		return fmt.Errorf("failed to configure secp256k1: %w", err)
	}

	if err := ri.runCommand("make"); err != nil {
		return fmt.Errorf("failed to build secp256k1: %w", err)
	}

	if err := ri.runCommand("sudo", "make", "install"); err != nil {
		return fmt.Errorf("failed to install secp256k1: %w", err)
	}

	if err := ri.runCommand("sudo", "ldconfig"); err != nil && runtime.GOOS == "linux" {
		return fmt.Errorf("failed to run ldconfig: %w", err)
	}

	return nil
}

func (ri *RelayInstaller) InstallKhatru() error {
	khatruDir := filepath.Join(ri.workDir, "khatru")

	if err := ri.runCommand("git", "clone", "https://github.com/fiatjaf/khatru.git", khatruDir); err != nil {
		return fmt.Errorf("failed to clone khatru: %w", err)
	}

	if err := os.Chdir(khatruDir); err != nil {
		return err
	}

	if err := ri.runCommand("go", "mod", "tidy"); err != nil {
		return fmt.Errorf("failed to tidy khatru: %w", err)
	}

	binPath := filepath.Join(ri.installDir, "khatru")
	if err := ri.runCommand("go", "build", "-o", binPath, "."); err != nil {
		return fmt.Errorf("failed to build khatru: %w", err)
	}

	return nil
}

func (ri *RelayInstaller) InstallRelayer() error {
	relayerDir := filepath.Join(ri.workDir, "relayer")

	if err := ri.runCommand("git", "clone", "https://github.com/fiatjaf/relayer.git", relayerDir); err != nil {
		return fmt.Errorf("failed to clone relayer: %w", err)
	}

	if err := os.Chdir(relayerDir); err != nil {
		return err
	}

	if err := ri.runCommand("go", "mod", "tidy"); err != nil {
		return fmt.Errorf("failed to tidy relayer: %w", err)
	}

	binPath := filepath.Join(ri.installDir, "relayer")
	if err := ri.runCommand("go", "build", "-o", binPath, "."); err != nil {
		return fmt.Errorf("failed to build relayer: %w", err)
	}

	return nil
}

func (ri *RelayInstaller) InstallStrfry() error {
	strfryDir := filepath.Join(ri.workDir, "strfry")

	if err := ri.runCommand("git", "clone", "https://github.com/hoytech/strfry.git", strfryDir); err != nil {
		return fmt.Errorf("failed to clone strfry: %w", err)
	}

	if err := os.Chdir(strfryDir); err != nil {
		return err
	}

	if err := ri.runCommand("git", "submodule", "update", "--init"); err != nil {
		return fmt.Errorf("failed to init submodules: %w", err)
	}

	if err := ri.runCommand("make", "setup-golpe"); err != nil {
		return fmt.Errorf("failed to setup golpe: %w", err)
	}

	if err := ri.runCommand("make"); err != nil {
		return fmt.Errorf("failed to build strfry: %w", err)
	}

	srcBin := filepath.Join(strfryDir, "strfry")
	dstBin := filepath.Join(ri.installDir, "strfry")
	if err := ri.runCommand("cp", srcBin, dstBin); err != nil {
		return fmt.Errorf("failed to copy strfry binary: %w", err)
	}

	return nil
}

func (ri *RelayInstaller) InstallRustRelay() error {
	rustRelayDir := filepath.Join(ri.workDir, "nostr-rs-relay")

	if err := ri.runCommand("git", "clone", "https://github.com/scsibug/nostr-rs-relay.git", rustRelayDir); err != nil {
		return fmt.Errorf("failed to clone rust relay: %w", err)
	}

	if err := os.Chdir(rustRelayDir); err != nil {
		return err
	}

	if err := ri.runCommand("cargo", "build", "--release"); err != nil {
		return fmt.Errorf("failed to build rust relay: %w", err)
	}

	srcBin := filepath.Join(rustRelayDir, "target", "release", "nostr-rs-relay")
	dstBin := filepath.Join(ri.installDir, "nostr-rs-relay")
	if err := ri.runCommand("cp", srcBin, dstBin); err != nil {
		return fmt.Errorf("failed to copy rust relay binary: %w", err)
	}

	return nil
}

func (ri *RelayInstaller) VerifyInstallation() error {
	if ri.skipVerify {
		return nil
	}

	binaries := []string{"khatru", "relayer", "strfry", "nostr-rs-relay"}

	for _, binary := range binaries {
		binPath := filepath.Join(ri.installDir, binary)
		if _, err := os.Stat(binPath); os.IsNotExist(err) {
			return fmt.Errorf("binary %s not found at %s", binary, binPath)
		}

		if err := ri.runCommand("chmod", "+x", binPath); err != nil {
			return fmt.Errorf("failed to make %s executable: %w", binary, err)
		}
	}

	return nil
}

func (ri *RelayInstaller) commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func (ri *RelayInstaller) runCommand(name string, args ...string) error {
	if name == "curl" && len(args) > 0 && strings.Contains(strings.Join(args, " "), "|") {
		fullCmd := fmt.Sprintf("%s %s", name, strings.Join(args, " "))
		cmd := exec.Command("bash", "-c", fullCmd)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (ri *RelayInstaller) InstallSecp256k1Only() error {
	fmt.Println("Installing secp256k1 library...")

	if err := os.MkdirAll(ri.workDir, 0755); err != nil {
		return err
	}

	if err := ri.installSecp256k1(); err != nil {
		return fmt.Errorf("failed to install secp256k1: %w", err)
	}

	fmt.Println("secp256k1 installed successfully")
	return nil
}

func (ri *RelayInstaller) InstallAll() error {
	fmt.Println("Detecting dependencies...")
	if err := ri.DetectDependencies(); err != nil {
		return err
	}

	fmt.Println("Installing missing dependencies...")
	if err := ri.InstallMissingDependencies(); err != nil {
		return err
	}

	if err := os.MkdirAll(ri.workDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(ri.installDir, 0755); err != nil {
		return err
	}

	fmt.Println("Installing khatru...")
	if err := ri.InstallKhatru(); err != nil {
		return err
	}

	fmt.Println("Installing relayer...")
	if err := ri.InstallRelayer(); err != nil {
		return err
	}

	fmt.Println("Installing strfry...")
	if err := ri.InstallStrfry(); err != nil {
		return err
	}

	fmt.Println("Installing rust relay...")
	if err := ri.InstallRustRelay(); err != nil {
		return err
	}

	fmt.Println("Verifying installation...")
	if err := ri.VerifyInstallation(); err != nil {
		return err
	}

	fmt.Println("All relays installed successfully")
	return nil
}
