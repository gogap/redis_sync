package main

import (
	"bytes"
	"os/exec"
	"strings"
)

type GitRepo struct {
	LastError error
	Output    []byte
}

func (p *GitRepo) Init() error {
	return p.run("init")
}

func (p *GitRepo) Status() error {
	return p.run("status", "--short")
}

func (p *GitRepo) Diff() error {
	return p.run("diff")
}

func (p *GitRepo) Add(files ...string) error {
	return p.run("add", files...)
}

func (p *GitRepo) Clean() error {
	return p.run("clean", "-d", "-x", "-f", "-q")
}

func (p *GitRepo) StashSaveAll() error {
	return p.run("stash", "-a")
}

func (p *GitRepo) StashApply() error {
	return p.run("stash", "apply")
}

func (p *GitRepo) StashPop() error {
	return p.run("stash", "pop")
}

func (p *GitRepo) StashDrop() error {
	return p.run("stash", "drop", "-q")
}

func (p *GitRepo) Commit(message string) error {
	return p.run("commit", "-m", message)
}

func (p *GitRepo) AddUntracked() error {
	return p.addStdout("ls-files", "--others", "--exclude-standard")
}

func (p *GitRepo) AddModified() error {
	return p.addStdout("ls-files", "-m")
}

func (p *GitRepo) IsClean() bool {
	if e := p.run("diff", "--shortstat"); e != nil {
		return false
	}

	if len(p.Output) == 0 {
		return true
	}

	return false
}

func (p *GitRepo) addStdout(subcmd string, arg ...string) error {
	p.run(subcmd, arg...)
	if p.LastError != nil {
		return p.LastError
	}

	n := bytes.Index(p.Output, nil)
	files := strings.Split(string(p.Output[:n]), "\n")
	p.Add(files...)
	return p.LastError
}

func (p *GitRepo) run(subcmd string, arg ...string) error {
	arg = prependArg(subcmd, arg)
	cmd := exec.Command("git", arg...)
	p.Output, p.LastError = cmd.CombinedOutput()
	return p.LastError
}

func prependArg(pre string, arg []string) []string {
	buffer := make([]string, 1)
	buffer[0] = pre
	return append(buffer, arg...)
}
