package scan

type Result interface {
	GetRegion() string
	Resources() []interface{}
}

type ErrorResult struct {
	ErrorString string
	Region string
}

func (r *ErrorResult) Error() string {
	return r.ErrorString
}

func (r *ErrorResult) GetRegion() string {
	return r.Region
}

func (r *ErrorResult) Resources() []interface{} {
	resources := make([]interface{}, 0)
	return resources
}

type ResourcesResult struct {
	Region string
	resources []interface{}
}

func NewResourcesResult() *ResourcesResult {
	resources := make([]interface{}, 0)
	return &ResourcesResult{resources: resources}
}

func (r *ResourcesResult) GetRegion() string {
	return r.Region
}

func (r *ResourcesResult) Resources() []interface{} {
	return r.resources
}

func (r *ResourcesResult) AddResource(resource interface{}) {
	r.resources = append(r.resources, resource)
	return
}
