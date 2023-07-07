package exp

import "errors"

var ErrUnknownType = errors.New("unknown panic type")

func Try(tryFunc func()) (retErr error) {
	defer func() {
		if e := recover(); e != nil {
			switch ee := e.(type) {
			case error:
				retErr = ee
			default:
				retErr = ErrUnknownType
			}
		}
	}()
	tryFunc()
	return nil
}

func RecoverOnPanic(f func()) {
	if e := recover(); e != nil {
		f()
	}
}

func ThrowOnError(err error) {
	if err != nil {
		Throw(err)
	}
}

func Throw(err error) {
	panic(err)
}

func TryFallback[T any](tryFunc func() T, fallback T) (retVal T) {
	defer func() {
		if e := recover(); e != nil {
			switch e.(type) {
			case error:
				retVal = fallback
			}
		}
	}()
	return tryFunc()
}
