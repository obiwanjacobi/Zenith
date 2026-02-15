package compiler

import "fmt"

func OfType[T interface{}, S any](collection []S) []T {
	result := make([]T, 0)
	for _, item := range collection {
		if typedItem, ok := any(item).(T); ok {
			result = append(result, typedItem)
		}
	}
	return result
}

// Go is very liberal in interpreting implemented interfaces and may return unexpected results.
// This function first selects the correct concrete struct type (T)
// and then checks if it implements the desired interface (I).
func OfTypeInterface[T any, I interface{}, S any](collection []S) []I {
	result := make([]I, 0)
	for _, item := range collection {
		if typedItem, ok := any(item).(T); ok {
			interfaceItem, ok := any(typedItem).(I)
			if !ok {
				panic(fmt.Sprintf("OfTypeInterface: item of type %T does not implement interface %T", typedItem, (*I)(nil)))
			}
			result = append(result, interfaceItem)
		}
	}
	return result
}
