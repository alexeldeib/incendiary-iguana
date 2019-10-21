package stringslice

// Has returns true if a given slice has the provided string s.
func Has(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// Add returns a []string with s appended if it is not already found in the provided slice.
func Add(slice []string, s string) []string {
	for _, item := range slice {
		if item == s {
			return slice
		}
	}
	return append(slice, s)
}

// Remove returns a newly created []string that contains all items from slice that are not equal to s.
func Remove(slice []string, s string) []string {
	keep := func(val string) bool {
		return val != s
	}
	return Filter(slice, keep)
}

// Filter reslices an array in place using a provided filter function.
func Filter(slice []string, keep func(val string) bool) []string {
	index := 0
	for _, item := range slice {
		if keep(item) {
			slice[index] = item
			index++
		}
	}
	return slice[:index]
}
