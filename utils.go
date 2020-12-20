package main

func StringArrayContains(array []string, val string) bool {
	for _, item := range array {
		if item == val {
			return true
		}
	}
	return false
}
