package storage

type Storage interface {
	// Get a document by key
	Get(string) ([]byte, error)

	// Delete a key
	Delete(string) error

	// Insert a document
	Insert(string, interface{}) error
}
