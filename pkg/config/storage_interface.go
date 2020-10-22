package config

// Storage represents an interface that shodan use to persist data.
// This is not really sophisticated but get the job done without requiring etcd.
type Storage interface {
	// Get accept a key name and return bytes stored under this key. The bytes are usually JSON string.
	// It returns StorageNotFound error in case there is nothing stored under the key.
	Get(string) ([]byte, error)

	// Delete will delete the object by the name provided.
	// It returns StorageNotFound error in case there is nothing stored under the key.
	// It trigger the storage informer in case the delete was success.
	Delete(string) error

	// List return a list of key names that have provided prefix. If the prefix is "" it returns ALL keys.
	List(string) ([]string, error)

	// Set persist the bytes provided in second parameter into a key given by first parameter.
	// It trigger the storage informer in case the create/update was success.
	Set(string, []byte) error
}
