package sync

type Report struct {
	Copied      int
	Overwritten int
	Deleted     int
	Skipped     int
	Errors      []error
}

func (r *Report) addErr(err error) {
	if err != nil {
		r.Errors = append(r.Errors, err)
	}
}
