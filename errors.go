package ptasks

type ErrJobs struct {
	Jobs []*Job
}

func NewErrJobs(jobs []*Job) *ErrJobs {
	return &ErrJobs{
		Jobs: jobs,
	}
}

func (e ErrJobs) Error() string {
	eTxt := ""
	for _, j := range e.Jobs {
		eTxt += j.err.Error() + "\n"
	}
	return eTxt
}

func IsErrErrJobs(err error) bool {
	_, ok := err.(*ErrJobs)
	return ok
}
