package libs

// checks if 2 string arrays contain the same elements
func StringArrayEqual(arr1, arr2 []string) bool {
	if len(arr1) != len(arr2) {
		return false
	}

	set := make(map[string]int, len(arr1))
	for i, value := range arr1 {
		set[value] = i
	}

	for _, value := range arr2 {
		if _, exist := set[value]; !exist {
			return false
		}
	}
	return true
}
