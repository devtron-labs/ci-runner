package helper

type CiStageError struct {
	Err error
}

func (err CiStageError) Error() string {
	return err.Err.Error()
}

func (err *CiStageError) Unwrap() error {
	return err.Err
}
