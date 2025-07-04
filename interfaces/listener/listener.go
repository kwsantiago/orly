package listener

type I interface {
	Write(p []byte) (n int, err error)
	Close() error
	Remote() string
}
