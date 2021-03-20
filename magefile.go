//+build mage

package main

import (
	"os"
	"os/exec"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	srcPath  = "./..."
	toolPath = "_tools/"
	binPath  = toolPath + "bin/"
)

var (
	pwd, _ = os.Getwd()
	env    = map[string]string{
		"GOBIN": pwd + "/" + binPath,
	}
)

// Build runs go mod download and then builds a local copy.
func Build() error {
	if err := sh.RunV("go", "mod", "download"); err != nil {
		return err
	}
	return sh.RunV("go", "build", "-o", "./bin/flipt", "./cmd/flipt")
}

// Clean cleans up.
func Clean() error {
	if err := sh.RunV("go", "clean", "-i", srcPath); err != nil {
		return err
	}

	if err := sh.RunV("packr", "clean"); err != nil {
		return err
	}

	if err := os.RemoveAll("dist"); err != nil {
		return err
	}

	return sh.RunV("go", "mod", "tidy")
}

// Fmt formats all go files.
func Fmt() error {
	if err := sh.RunV("gofmt", "-w", "-s", "."); err != nil {
		return err
	}

	return sh.RunV("goimports", "-w", ".")
}

// Test runs all the tests.
func Test() error {
	sourceFiles := os.Getenv("SOURCE_FILES")
	if sourceFiles == "" {
		sourceFiles = srcPath
	}

	testPattern := os.Getenv("TEST_PATTERN")
	if testPattern == "" {
		testPattern = "."
	}

	return sh.RunV("go", "test", "-covermode=atomic", "-count=1", "-coverprofile=coverage.txt", sourceFiles, "-run="+testPattern, "-timeout=30s", "-v")
}

// Cover runs all the tests and opens the coverage report.
func Cover() error {
	mg.Deps(Test)
	return sh.RunV("go", "tool", "cover", "-html=coverage.txt")
}

//  Pack packs the assets in the binary.
func Pack() error {
	return sh.RunWithV(env, "packr", "-i", "cmd/flipt")
}

// Lint runs all the linters.
func Lint() error {
	return sh.RunWithV(env, "golangci-lint", "run")
}

var tools = []string{
	"github.com/golang/protobuf/protoc-gen-go@v1.4.2",
	"github.com/gobuffalo/packr/packr",
	"google.golang.org/grpc/cmd/protoc-gen-go-grpc",
	"github.com/golangci/golangci-lint/cmd/golangci-lint",
	"github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway",
	"github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger",
	"golang.org/x/tools/cmd/cover",
	"golang.org/x/tools/cmd/goimports",
	"google.golang.org/grpc",
	"github.com/buchanae/github-release-notes",
}

// Bootstrap installs all tools required to build.
func Bootstrap() error {
	if err := os.MkdirAll(binPath, os.ModePerm); err != nil {
		return err
	}

	if err := os.Chdir(toolPath); err != nil {
		return err
	}

	_, err := os.Stat("go.mod")
	if err != nil {
		if os.IsNotExist(err) {
			if err := sh.RunV("go", "mod", "init", "tools"); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	for _, pkg := range tools {
		if err := sh.RunWithV(env, "go", "get", "-u", "-v", pkg); err != nil {
			return err
		}
	}
	return nil
}

func isCmdAvailable(cmd string) (bool, error) {
	_, err := exec.LookPath(cmd)
	if err != nil {
		return false, nil
	}
	return true, nil
}
