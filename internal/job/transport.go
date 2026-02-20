package job

import "github.com/ahmethakanbesel/finance-api/internal/apperror"

type GetJobRequest struct {
	ID int64
}

func (r GetJobRequest) Validate() *apperror.AppError {
	if r.ID <= 0 {
		return apperror.New(apperror.BadRequest, "invalid job id")
	}
	return nil
}

type ListJobsRequest struct {
	Source string
	Symbol string
}

func (r ListJobsRequest) Validate() *apperror.AppError {
	return nil
}
