//go:build ignore
// +build ignore

/*
 * Copyright (c) 2021 The GoPlus Authors (goplus.org). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/goplus/gop/env"
)

func getcwd() string {
	path, _ := os.Getwd()
	return path
}

func getGopLocalLink() string {
	path, _ := os.UserHomeDir()
	return filepath.Join(path, "gop")
}

func checkPathExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

var gopRoot = getcwd()
var gopLocalLink = getGopLocalLink()
var initCommandExecuteEnv = os.Environ()
var commandExecuteEnv = initCommandExecuteEnv

func execCommand(command string, arg ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(command, arg...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = commandExecuteEnv
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func getBuildBranch() string {
	branch, stderr, err := execCommand("git", "branch", "--show-current")
	if err != nil || stderr != "" {
		return ""
	}
	return strings.TrimRight(branch, "\n")
}

func getRevCommit(tag string) string {
	commit, stderr, err := execCommand("git", "rev-parse", "--verify", tag)
	if err != nil || stderr != "" {
		return ""
	}
	return strings.TrimRight(commit, "\n")
}

func getGitInfo() (string, bool) {
	gitDir := filepath.Join(gopRoot, ".git")
	if checkPathExist(gitDir) {
		return getRevCommit("HEAD"), true
	}
	return "", false
}

func getBuildDateTime() string {
	now := time.Now()
	return now.Format("2006-01-02_15-04-05")
}

func findTag(commit string) string {
	tagRet, tagErr, err := execCommand("git", "tag")
	if err != nil || tagErr != "" {
		return ""
	}
	var prefix = "v" + env.MainVersion + "."
	for _, tag := range strings.Split(tagRet, "\n") {
		if strings.HasPrefix(tag, prefix) {
			if getRevCommit(tag) == commit {
				return tag
			}
		}
	}
	return ""
}

func getGopBuildFlags() string {
	buildFlags := fmt.Sprintf("-X github.com/goplus/gop/env.defaultGopRoot=%s", gopRoot)
	buildFlags = fmt.Sprintf("-X github.com/goplus/gop/env.buildDate=%s", getBuildDateTime())
	if commit, ok := getGitInfo(); ok {
		buildFlags += fmt.Sprintf(" -X github.com/goplus/gop/env.buildCommit=%s", commit)
		if branch := getBuildBranch(); branch != "" {
			buildFlags += fmt.Sprintf(" -X github.com/goplus/gop/env.buildBranch=%s", branch)
		}
		if buildVer := findTag(commit); buildVer != "" {
			buildFlags += fmt.Sprintf(" -X github.com/goplus/gop/env.buildVersion=%s", buildVer)
		}
	}
	return buildFlags
}

func detectGoBinPath() string {
	goBin, ok := os.LookupEnv("GOBIN")
	if ok {
		return goBin
	}

	goPath, ok := os.LookupEnv("GOPATH")
	if ok {
		return filepath.Join(goPath, "bin")
	}

	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, "go", "bin")
}

func buildGoplusTools(useGoProxy bool) {
	commandsDir := filepath.Join(gopRoot, "cmd")
	if !checkPathExist(commandsDir) {
		println("Error: This script should be run at the root directory of gop repository.")
		os.Exit(1)
	}

	buildFlags := getGopBuildFlags()

	if useGoProxy {
		println("Info: we will use goproxy.cn as a Go proxy to accelerate installing process.")
		commandExecuteEnv = append(commandExecuteEnv,
			"GOPROXY=https://goproxy.cn,direct",
		)
	}

	println("Installing Go+ tools...")
	os.Chdir(commandsDir)
	buildOutput, buildErr, err := execCommand("go", "install", "-v", "-ldflags", buildFlags, "./...")
	println(buildErr)
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
	println(buildOutput)

	println("Go+ tools installed successfully!")
}

func linkGoplusToLocal() {
	fmt.Printf("Linking %s to %s\n", gopRoot, gopLocalLink)

	os.Chdir(gopRoot)
	if gopLocalLink != gopRoot && !checkPathExist(gopLocalLink) {
		err := os.Symlink(gopRoot, gopLocalLink)
		if err != nil {
			println(err.Error())
		}
	}

	fmt.Printf("%s linked to %s successfully!\n", gopRoot, gopLocalLink)
}

func runTestcases() {
	println("Start running testcases.")
	os.Chdir(gopRoot)

	path, _ := os.LookupEnv("PATH")
	path = fmt.Sprintf("%s:", detectGoBinPath()) + path
	commandExecuteEnv = append(commandExecuteEnv, "PATH="+path)

	coverage := "-coverprofile=coverage.txt"
	testOutput, testErr, err := execCommand("gop", "test", coverage, "-covermode=atomic", "./...")
	println(testOutput)
	println(testErr)
	if err != nil {
		println(err.Error())
	}

	println("End running testcases.")
}

func localUninstall() {
	println("Uninstalling Go+ and related tools.")

	goBinPath := detectGoBinPath()
	filesToRemove := []string{
		gopLocalLink,
		filepath.Join(goBinPath, "gop"),
		filepath.Join(goBinPath, "gopfmt"),
		filepath.Join(goBinPath, "goptestgo"),
	}

	for _, file := range filesToRemove {
		if !checkPathExist(file) {
			continue
		}
		if err := os.Remove(file); err != nil {
			println(err.Error())
		}
	}

	println("Go+ and related tools uninstalled successfully.")
}

func isInChina() bool {
	const prefix = "LANG=\""
	out, errMsg, err := execCommand("locale")
	if err != nil || errMsg != "" {
		return false
	}
	if strings.HasPrefix(out, prefix) {
		out = out[len(prefix):]
		return strings.HasPrefix(out, "zh_CN") || strings.HasPrefix(out, "zh_HK")
	}
	return false
}

func main() {
	isInstall := flag.Bool("install", false, "Install Go+")
	isBuild := flag.Bool("build", false, "Build the Go+")
	isTest := flag.Bool("test", false, "Run testcases")
	isUninstall := flag.Bool("uninstall", false, "Uninstall Go+")
	isGoProxy := flag.Bool("proxy", false, "Set GOPROXY for people in China")
	isAutoProxy := flag.Bool("autoproxy", false, "Check to set GOPROXY automatically")

	flag.Parse()

	useGoProxy := *isGoProxy
	if !useGoProxy && *isAutoProxy {
		useGoProxy = isInChina()
	}
	flagActionMap := map[*bool]func(){
		isInstall: func() {
			buildGoplusTools(useGoProxy)
			linkGoplusToLocal()
			println("Go+ is now installed.")
		},
		isTest:      runTestcases,
		isBuild:     func() { buildGoplusTools(useGoProxy) },
		isUninstall: localUninstall,
	}

	for flag, action := range flagActionMap {
		if *flag {
			action()
			return
		}
	}

	println("Usage:\n")
	flag.PrintDefaults()
}