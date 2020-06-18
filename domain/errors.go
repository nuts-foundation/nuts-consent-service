package domain

import "errors"

var ErrAggregateCancelled = errors.New("aggregate cancelled")
var ErrUnknownCommand = errors.New("unknown command")
