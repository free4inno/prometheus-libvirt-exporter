package collector

import "errors"

// ErrNoData indicates the collector found no data to collect, but had no other error.
var ErrNoData = errors.New("collector returned no data")

func IsNoDataError(err error) bool {
	return err == ErrNoData
}

// ErrNotProvided indicates the collector was not provided with the necessary data to collect.
var ErrNotProvided = errors.New("collector not provided with necessary data")

func IsNotProvidedError(err error) bool {
	return err == ErrNotProvided
}
