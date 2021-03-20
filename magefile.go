//+build mage

package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const GOBIN = "_tools/bin/"

var env = map[string]string{
	"GOBIN": GOBIN,
}

// Buld runs go mod download and then builds a local copy.
func Build() error {
	if err := sh.RunV("go", "mod", "download"); err != nil {
		return err
	}
	return sh.RunV("go", "build", "-o", "./bin/flipt", "./cmd/flipt")
}

// Test runs all the tests.
func Test() error {
	sourceFiles := os.Getenv("SOURCE_FILES")
	if sourceFiles == "" {
		sourceFiles = "./..."
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

//  Pack the assets in the binary.
func Pack() error {
	ok, err := isCmdAvailable(GOBIN + "packr")
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("could not find %q in path, may need to run bootstrap", "packr")
	}

	return sh.RunV(GOBIN+"packr", "-i", "cmd/flipt")
}

// Lint runs all the linters.
func Lint() error {
	ok, err := isCmdAvailable(GOBIN + "golangci-lint")
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("could not find %q in path, may need to run bootstrap", "golangci-lint")
	}

	return sh.RunV(GOBIN+"golangci-lint", "run")
}

const tools = `
    "github.com/gobuffalo/packr/packr"
    "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
    "github.com/golangci/golangci-lint/cmd/golangci-lint"
    "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway"
    "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger"
    "golang.org/x/tools/cmd/cover"
    "golang.org/x/tools/cmd/goimports"
    "google.golang.org/grpc"
    "github.com/buchanae/github-release-notes"`

// Bootstrap all tools required to build.
func Bootstrap() error {
	if err := os.MkdirAll(GOBIN, os.FileMode(os.O_RDWR)); err != nil {
		return err
	}

	if err := os.Chdir(GOBIN); err != nil {
		return err
	}

	_, err := os.Stat("go.mod")
	if os.IsNotExist(err) {
		if err := sh.RunV("go", "mod", "init", "tools"); err != nil {
			return err
		}
	} else {
		return err
	}

	return sh.RunWithV(env, "go", "get", "-u", "-v", "github.com/golang/protobuf/protoc-gen-go@v1.4.2")
}

func isCmdAvailable(cmd string) (bool, error) {
	_, err := exec.LookPath(cmd)
	if err != nil {
		return false, nil
	}
	return true, nil
}
