package models

type UnauthorizedError struct {
	underlying error
}

type BadRequestError struct {
	underlying error
}

type SdkErr struct {
	underlying error
}

type NotFoundError struct {
	underlying error
}

func (ue *UnauthorizedError) Error() string {
	return ue.underlying.Error()
}

func (br *BadRequestError) Error() string {
	return br.underlying.Error()
}

func (se *SdkErr) Error() string {
	return se.underlying.Error()
}

func (nfe *NotFoundError) Error() string {
	return nfe.underlying.Error()
}

func NewSdkErr(underlying error) *SdkErr {
	return &SdkErr{underlying: underlying}
}

func NewUnauthorizedError(underlying error) *UnauthorizedError {
	return &UnauthorizedError{underlying: underlying}
}

func NewBadRequestError(underlying error) *BadRequestError {
	return &BadRequestError{underlying: underlying}
}

func NewNotFoundError(underlying error) *NotFoundError {
	return &NotFoundError{underlying: underlying}
}
