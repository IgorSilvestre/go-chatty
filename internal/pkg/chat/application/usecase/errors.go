package usecase

import "fmt"

// ErrPersistence indicates an infrastructure/repository failure inside a use case
var ErrPersistence = fmt.Errorf("chat use case persistence error")
