package storage

// Storage is an abstraction over database engines
type Storage interface {
	GetName() string
	ListStates() (states []string, err error)
	GetState(name string, serial int) (document interface{}, err error)
	ListStateSerials(name string) (serials []string, err error)
	InsertState(document interface{}, timestamp, source, name string) (err error)
	RemoveState(name string) (err error)
	GetLockStatus(name string) (lockStatus interface{}, err error)
	LockState(name string, lockData interface{}) (err error)
	UnlockState(name string, lockData interface{}) (err error)
}
