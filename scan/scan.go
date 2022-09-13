package scan

type Scan interface {
	Scan() ([]interface{}, error)
}
