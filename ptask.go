package ptasks

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/logrusorgru/aurora"
	"github.com/mattn/go-isatty"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	JobStateWait jobState = 1 << iota
	JobStateRun
	JobStateFinish
)

type jobState int

type Job struct {
	Name  string
	Data  interface{}
	state jobState
	err   error
	buf   *bytes.Buffer
	color aurora.Aurora
}

func NewJob(name string, data interface{}) *Job {
	return &Job{
		Name:  name,
		Data:  data,
		state: JobStateWait,
		buf:   &bytes.Buffer{},
		color: aurora.NewAurora(false),
	}
}

func (j *Job) Buffer() *bytes.Buffer {
	return j.buf
}

func (j *Job) SetError(e error) {
	j.err = e
}

func (j *Job) Header() string {
	header := fmt.Sprintf("Task %s ", j.color.Cyan(j.Name))
	if j.state == JobStateWait {
		header += fmt.Sprintf("[%s]", j.color.BrightYellow("Waiting"))
	} else if j.state == JobStateRun {
		header += fmt.Sprintf("[%s]", j.color.BrightBlue("Running"))
	} else if j.state == JobStateFinish && j.err != nil {
		header += fmt.Sprintf("[%s]     ", j.color.Red("Fail"))
	} else {
		header += fmt.Sprintf("[%s]     ", j.color.Green("Ok"))
	}
	return header
}

func (j *Job) DrawOutput() string {
	d := j.Header() + "\n"

	d += fmt.Sprintf("├─── [%s]\n│\n", j.color.Blue("Output"))

	scanner := bufio.NewScanner(j.buf)
	for scanner.Scan() {
		d += fmt.Sprintf("│  %s\n", scanner.Text())
	}

	if j.err != nil {
		d += "│\n"
		d += fmt.Sprintf("├─── [%s]\n│\n", j.color.Red("Error"))
		scanner := bufio.NewScanner(bytes.NewBufferString(j.err.Error()))
		for scanner.Scan() {
			d += fmt.Sprintf("│  %s\n", scanner.Text())
		}
		d += "└───\n"
	} else {
		d += "└───\n"
	}

	return d
}

type Ptask struct {
	output      io.Writer
	isatty      bool
	workerFunc  func(job *Job)
	color       aurora.Aurora
	compact     bool
	termHeight  int
	termWidth   int
	onlyErrors  bool
	noVerbose   bool
	notDrawable bool
	noHeader    bool
}

type optFunc func(pt *Ptask)

// only show output from errored task
func OnlyErrorsOpt(onlyErrors bool) optFunc {
	return func(pt *Ptask) {
		pt.onlyErrors = onlyErrors
	}
}

// do not show output from task
func NoVerboseOpt(noVerbose bool) optFunc {
	return func(pt *Ptask) {
		pt.noVerbose = noVerbose
	}
}

// Do not see details from each task, only set fail or ok in output
func CompactOpt(compact bool) optFunc {
	return func(pt *Ptask) {
		pt.compact = compact
	}
}

// Force to use a tty for getting color output
func ForceTtyOpt(forceTty bool) optFunc {
	return func(pt *Ptask) {
		if forceTty {
			pt.isatty = forceTty
		}
	}
}

// Do not draw tasks for refreshing their state in output
// This will only output details when all tasks complete
func NotDrawableOpt(notDrawable bool) optFunc {
	return func(pt *Ptask) {
		pt.notDrawable = notDrawable
	}
}

// Do not write `Running all tasks in parallel with...` in output
func NotHeaderOpt(noHeader bool) optFunc {
	return func(pt *Ptask) {
		pt.noHeader = noHeader
	}
}

// see details on other optFunc
func AllInOneOpt(onlyErrors, noVerbose, compact, forceTty, notDrawable, noHeader bool) optFunc {
	return func(pt *Ptask) {
		pt.noHeader = noHeader
		pt.onlyErrors = onlyErrors
		pt.noVerbose = noVerbose
		pt.compact = compact
		if forceTty {
			pt.isatty = forceTty
		}
		pt.notDrawable = notDrawable
	}
}

func NewPtask(output io.Writer, workerFunc func(*Job), opts ...optFunc) *Ptask {
	tty := false
	height := 0
	width := 0
	if f, ok := output.(*os.File); ok {
		if !tty {
			tty = isatty.IsTerminal(f.Fd())
		}
		tmpWidth, tmpHeight, err := terminal.GetSize(int(os.Stdout.Fd()))
		if err == nil {
			height = tmpHeight
			width = tmpWidth
		}
	}

	t := &Ptask{
		output:     output,
		workerFunc: workerFunc,
		isatty:     tty,
		color:      aurora.NewAurora(tty),
		termHeight: height,
		termWidth:  width,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

func (c *Ptask) isDrawable(nbJobs int) bool {
	return c.isatty && c.termHeight >= nbJobs && !c.notDrawable
}

func (c *Ptask) sizeMaxName(jobs []*Job) int {
	max := 0
	for _, j := range jobs {
		if len(j.Name) > max {
			max = len(j.Name)
		}
	}
	return max
}

func (c *Ptask) resizeJobName(sizeMaxName int, jobs []*Job) {
	if sizeMaxName-20 > c.termWidth {
		return
	}
	for _, j := range jobs {
		nbSpace := sizeMaxName - len(j.Name)
		if nbSpace <= 0 {
			continue
		}
		for i := 0; i < nbSpace; i++ {
			j.Name += " "
		}
	}
}

func (c *Ptask) Run(jobs []*Job, parallel int) error {
	if !c.noHeader {
		fmt.Fprintf(c.output, "Running all tasks in parallel with %d workers ... ", parallel)
		if !c.compact {
			fmt.Fprint(c.output, "\n")
		}
	}
	c.resizeJobName(c.sizeMaxName(jobs), jobs)
	nbJobs := len(jobs)
	jobsChan := make(chan *Job, nbJobs)
	for _, j := range jobs {
		j.color = c.color
	}
	// hide cursor when on terminal
	if c.isDrawable(nbJobs) {
		c.CompactPrintln("\033[?25l")
		// handling re-show cursor even on interruption
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGKILL, syscall.SIGQUIT, syscall.SIGSTOP)
		go func() {
			sig := <-sigs
			sig.Signal()

			c.CompactPrintln("\033[?25h")
			os.Exit(1)
		}()
		c.drawJobs(jobs)
	}

	for i := 0; i < parallel; i++ {
		go c.worker(jobsChan)
	}
	for _, j := range jobs {
		jobsChan <- j
	}
	hasErr := false
	for {
		nbFinished := 0
		for _, j := range jobs {
			if j.state == JobStateFinish {
				nbFinished++
			}
			if j.err != nil {
				hasErr = true
			}
		}
		if c.isDrawable(nbJobs) {
			c.drawJobs(jobs)
		}
		if nbFinished == nbJobs {
			// re-show cursor when on terminal
			if c.isDrawable(nbJobs) {
				c.CompactPrint("\033[%dB", nbJobs)
				c.CompactPrint("\033[?25h")
			}
			break
		}
		if c.isDrawable(nbJobs) {
			time.Sleep(50 * time.Millisecond)
		} else {
			time.Sleep(1 * time.Second)
		}

	}
	if !c.isDrawable(nbJobs) {
		c.drawJobs(jobs)
	}
	c.CompactPrintln("")
	if !c.noVerbose && !c.compact {
		c.showDetails(jobs, hasErr)
	}
	if !hasErr {
		if c.compact {
			fmt.Fprintf(c.output, "[%s]\n", c.color.Green("Ok"))
		}
		return nil
	}
	if c.compact {
		fmt.Fprintf(c.output, "[%s]\n", c.color.Red("Fail"))
	}
	eJobs := make([]*Job, 0)
	for _, j := range jobs {
		if j.err == nil {
			continue
		}
		eJobs = append(eJobs, j)
	}
	return NewErrJobs(eJobs)
}

func (c *Ptask) drawJobs(jobs []*Job) {
	if c.compact {
		return
	}
	for _, j := range jobs {
		fmt.Fprint(c.output, j.Header()+"\n")
	}

	if c.isDrawable(len(jobs)) {
		c.CompactPrint("\033[%dA", len(jobs))
	}
}

func (c *Ptask) showDetails(jobs []*Job, hasErr bool) {
	if c.onlyErrors && !hasErr {
		return
	}
	if !c.onlyErrors {
		fmt.Fprintf(c.output, `
┌───────────────┐
│  All outputs  │
└───────────────┘
`)
	} else {
		fmt.Fprintf(c.output, `
┌──────────────────┐
│  %s outputs  │
└──────────────────┘
`, c.color.Red("Errors"))
	}

	for _, j := range jobs {
		if c.onlyErrors && j.err == nil {
			continue
		}
		fmt.Fprintf(c.output, j.DrawOutput())
	}
}

func (c *Ptask) worker(jobs <-chan *Job) {
	for j := range jobs {
		j.state = JobStateRun
		c.workerFunc(j)
		j.state = JobStateFinish
	}
}

func (c *Ptask) CompactPrintln(f string, a ...interface{}) {
	if c.compact {
		return
	}
	fmt.Fprintf(c.output, f+"\n", a...)
}

func (c *Ptask) CompactPrint(f string, a ...interface{}) {
	if c.compact {
		return
	}
	fmt.Fprintf(c.output, f, a...)
}
