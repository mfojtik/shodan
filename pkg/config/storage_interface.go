package config

type Storage interface {
	Get(string) ([]byte, error)
	Delete(string) error
	List(string) ([]string, error)
	Set(string, []byte) error
}
