package komut_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPOSIXLauncherSelectsBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX launcher test")
	}

	tests := []struct {
		osName string
		arch   string
		target string
	}{
		{osName: "Darwin", arch: "arm64", target: "darwin-arm64"},
		{osName: "Darwin", arch: "x86_64", target: "darwin-amd64"},
		{osName: "Linux", arch: "aarch64", target: "linux-arm64"},
		{osName: "Linux", arch: "x86_64", target: "linux-amd64"},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			root := makeLauncherFixture(t, tt.target)
			fakeBin := filepath.Join(root, "fake-bin")
			mustMkdirAll(t, fakeBin)
			uname := filepath.Join(fakeBin, "uname")
			mustWriteExecutable(t, uname, "#!/bin/sh\ncase \"$1\" in\n-s) echo \"$FAKE_OS\" ;;\n-m) echo \"$FAKE_ARCH\" ;;\nesac\n")

			cmd := exec.Command(filepath.Join(root, "bin", "x"))
			cmd.Env = append(os.Environ(), "PATH="+fakeBin+":"+os.Getenv("PATH"), "FAKE_OS="+tt.osName, "FAKE_ARCH="+tt.arch)
			output, err := cmd.Output()
			if err != nil {
				t.Fatal(err)
			}
			if string(output) != tt.target+"\n" {
				t.Fatalf("launcher output = %q", output)
			}
		})
	}
}

func makeLauncherFixture(t *testing.T, target string) string {
	t.Helper()
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "bin"))
	launcher, err := os.ReadFile(filepath.Join("bin", "x"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "x"), launcher, 0o755); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(root, "libexec", "x", target, "x")
	mustMkdirAll(t, filepath.Dir(targetPath))
	mustWriteExecutable(t, targetPath, "#!/bin/sh\necho "+target+"\n")
	return root
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWriteExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}
