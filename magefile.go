//+build mage

package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/magefile/mage/target"
)

const (
	srcPath           = "./..."
	dot               = "."
	toolPath          = "_tools/"
	binPath           = toolPath + "bin/"
	uiPath            = "ui/"
	uiSourcePath      = uiPath + "src/"
	uiDistPath        = uiPath + "dist/"
	uiNodeModulesPath = uiPath + "/node_modules/"
)

var (
	pwd, _ = os.Getwd()
	env    = map[string]string{
		"GOBIN": pwd + "/" + binPath,
		"PATH":  "$PATH:" + pwd + "/" + binPath,
	}

	tools = []string{
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

	Default = Build
)

// Bench runs all the benchmarks.
func Bench() error {
	sourceFiles := os.Getenv("SOURCE_FILES")
	if sourceFiles == "" {
		sourceFiles = srcPath
	}

	benchPattern := os.Getenv("BENCH_PATTERN")
	if benchPattern == "" {
		benchPattern = dot
	}

	return sh.RunV("go", "test", "-bench="+benchPattern, sourceFiles, "-run=XXX", "-v")
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

// Build runs go mod download and then builds a local copy.
func Build() error {
	mg.Deps(UI, Pack)

	if err := sh.RunV("go", "mod", "download"); err != nil {
		return err
	}

	return sh.RunV("go", "build", "-o", "./bin/flipt", "./cmd/flipt")
}

// Clean cleans up.
func Clean() error {
	fmt.Println("--> cleaning up..")

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

// Cover runs all the tests and opens the coverage report.
func Cover() error {
	mg.Deps(Test)
	fmt.Println("--> generating test coverage..")
	return sh.RunV("go", "tool", "cover", "-html=coverage.txt")
}

// Fmt formats all go files.
func Fmt() error {
	fmt.Println("--> formatting..")

	if err := sh.RunV("gofmt", "-w", "-s", dot); err != nil {
		return err
	}

	return sh.RunV("goimports", "-w", dot)
}

// Lint runs all the linters.
func Lint() error {
	fmt.Println("--> running linter..")
	return sh.RunWithV(env, "golangci-lint", "run")
}

//  Pack packs the assets in the binary.
func Pack() error {
	fmt.Println("--> packing assets..")
	return sh.RunWithV(env, "packr", "-i", "cmd/flipt")
}

// Proto generates protobufs.
func Proto() error {
	fmt.Println("--> generating protos..")
	return sh.RunWithV(env, "protoc",
		"-Irpc",
		"--go_out=./rpc",
		"--go-grpc_out=./rpc",
		"--grpc-gateway_out=logtostderr=true,grpc_api_configuration=./rpc/flipt.yaml:./rpc",
		"--swagger_out=logtostderr=true,grpc_api_configuration=./rpc/flipt.yaml:./swagger",
		"flipt.proto")
}

// Test runs all the tests.
func Test() error {
	fmt.Println("--> running tests..")
	sourceFiles := os.Getenv("SOURCE_FILES")
	if sourceFiles == "" {
		sourceFiles = srcPath
	}

	testPattern := os.Getenv("TEST_PATTERN")
	if testPattern == "" {
		testPattern = dot
	}

	return sh.RunV("go", "test", "-covermode=atomic", "-count=1", "-coverprofile=coverage.txt", sourceFiles, "-run="+testPattern, "-timeout=30s", "-v")
}

func uiDeps() error {
	fmt.Println("--> checking ui dependencies..")

	// if any ui deps changed, run yarn
	newer, err := target.Dir(uiNodeModulesPath, uiPath+"package.json", uiPath+"yarn.lock")
	if err != nil {
		return err
	}

	if newer {
		if err := os.Chdir(uiPath); err != nil {
			return err
		}

		defer os.Chdir("..")

		return sh.RunV("yarn", "--frozen-lockfile")
	}

	fmt.Println("  up to date")
	return nil
}

func UI() error {
	mg.Deps(Clean)

	if err := uiDeps(); err != nil {
		return err
	}

	fmt.Println("--> checking ui assets..")

	// if any ui sourcefiles change we need to run yarn build
	newer, err := target.Dir(uiDistPath, uiSourcePath)
	if err != nil {
		return err
	}

	if newer {
		if err := os.Chdir(uiPath); err != nil {
			return err
		}

		defer os.Chdir("..")

		return sh.RunV("yarn", "build")
	}

	fmt.Println("  up to date")
	return nil
}

func isCmdAvailable(cmd string) (bool, error) {
	_, err := exec.LookPath(cmd)
	if err != nil {
		return false, nil
	}
	return true, nil
}
