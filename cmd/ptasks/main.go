package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/ArthurHlt/ptasks"
	"github.com/jessevdk/go-flags"
	"github.com/mattn/go-isatty"
)

type Options struct {
	Verbose     bool           `short:"v" long:"verbose" description:"Set to pass in verbose mode."`
	OnlyErrors  bool           `short:"e" long:"only-errors" description:"See only error on verbose output."`
	ForceTty    bool           `short:"t" long:"tty" description:"Force use of tty for color output"`
	NotDrawable bool           `short:"d" long:"not-drawable" description:"Do not perform terminal draw"`
	NbWorker    int            `short:"n" long:"number-worker" default:"4" description:"number of worker to use"`
	Input       flags.Filename `short:"i" long:"file-input" description:"input to give to stdin in task, this must be a file path"`
	Compact     bool           `short:"c" long:"compact" description:"Set it to use compact form, this will not describe each script."`
}

var opts Options

type CmdTask struct {
	r   io.Reader
	cmd []string
}

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		log.Fatal(err.Error())
	}
	cmdRaw, _ := ioutil.ReadAll(os.Stdin)

	var stdinJob io.Reader = &bytes.Buffer{}
	if opts.Input != "" {
		f, err := os.Open(string(opts.Input))
		if err != nil {
			log.Fatal(err.Error())
		}
		stdinJob = f

	}
	mr := ptasks.NewFanoutReader(stdinJob)
	jobs := make([]*ptasks.Job, 0)
	cmds, err := ptasks.ParseCommands(string(cmdRaw))
	if err != nil {
		panic(err)
	}
	for _, cmd := range cmds {
		var r io.Reader
		var w io.Writer
		r, w = io.Pipe()
		mr.AddWriter(w)
		jobs = append(jobs, ptasks.NewJob(strings.Join(cmd, " "), CmdTask{
			r:   r,
			cmd: cmd,
		}))

	}
	go func() {
		err := mr.Fanout()
		if err != nil {
			log.Fatal(err.Error())
		}
	}()

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	isTTY := isatty.IsTerminal(os.Stdout.Fd())

	t := ptasks.NewPtask(os.Stdout, func(job *ptasks.Job) {
		task := job.Data.(CmdTask)
		err := RunProcess(ProcessOpts{
			Cmd:     task.cmd,
			EnvVars: os.Environ(),
			Stdout:  job.Buffer(),
			Stderr:  job.Buffer(),
			WithPty: opts.ForceTty || isTTY,
			Stdin:   task.r,
			WorkDir: wd,
		})
		if err != nil {
			job.SetError(err)
		}
	},
		ptasks.AllInOneOpt(opts.OnlyErrors, !opts.Verbose && !opts.OnlyErrors, opts.Compact, opts.ForceTty, opts.NotDrawable, false),
	)
	err = t.Run(jobs, opts.NbWorker)
	if err != nil {
		os.Exit(1)
	}
}
