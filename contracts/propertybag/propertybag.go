package propertybag

type IPropertyBag interface {
	// Get gets a value from the bag
	Get(key string) (any, bool)
	// Set sets a value in the bag
	Set(key string, value any)
	// Delete deletes a value from the bag
	Delete(key string)
	// Keys returns all keys in the bag
	Keys() []string
	// AsMap returns the bag as a map
	AsMap() map[string]any
}

type IRequestContextLoggingPropertyBag interface {
	IPropertyBag
}
