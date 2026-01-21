package compiler

func OfType[T interface{}, S any](collection []S) []T {
	result := make([]T, 0)
	for _, item := range collection {
		if typedItem, ok := any(item).(T); ok {
			result = append(result, typedItem)
		}
	}
	return result
}
