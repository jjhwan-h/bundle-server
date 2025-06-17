package errors

import "errors"

var (
	ErrEmptyEnvVar = errors.New("environment variable is empty")
	ErrBuildData   = errors.New("failed to build data.json")
	ErrEncodeData  = errors.New("failed to encoding data")
	ErrSaveData    = errors.New("failed to save data")
	ErrBuildBundle = errors.New("failed to build bundle")
)
