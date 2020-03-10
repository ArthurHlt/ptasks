# Ptasks

Lib for running tasks in parallel with nice output.

You can also use as a command line by doing:

```bash
go get github.com/ArthurHlt/ptasks/cmd/ptasks
```

You could now perform: 

```
$ cat myfilecontainingcmd.txt | ptasks -v
```

## Example of usage:

see [/cmd/ptasks/main.go](/cmd/ptasks/main.go)


subset:


```go
func main(){
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
}
```

## Command line ptasks

```
Usage:
  ptasks [OPTIONS]

Application Options:
  -v, --verbose        Set to pass in verbose mode.
  -e, --only-errors    See only error on verbose output.
  -t, --tty            Force use of tty for color output
  -d, --not-drawable   Do not perform terminal draw
  -n, --number-worker= number of worker to use (default: 4)
  -i, --file-input=    input to give to stdin in task, this must be a file path
  -c, --compact        Set it to use compact form, this will not describe each script.

Help Options:
  -h, --help           Show this help message

Usage:
  ptasks [OPTIONS]

Application Options:
  -v, --verbose        Set to pass in verbose mode.
  -e, --only-errors    See only error on verbose output.
  -t, --tty            Force use of tty for color output
  -d, --not-drawable   Do not perform terminal draw
  -n, --number-worker= number of worker to use (default: 4)
  -i, --file-input=    input to give to stdin in task, this must be a file path
  -c, --compact        Set it to use compact form, this will not describe each script.

Help Options:
  -h, --help           Show this help message
```