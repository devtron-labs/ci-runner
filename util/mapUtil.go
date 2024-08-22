package util

func MergeMaps(map1, map2 map[string][]string) map[string][]string {

	merged := make(map[string][]string)

	addEntries := func(source map[string][]string) {
		for key, values := range source {
			if _, exists := merged[key]; !exists {
				merged[key] = values
			} else {
				valueSet := make(map[string]bool)
				for _, v := range merged[key] {
					valueSet[v] = true
				}
				for _, v := range values {
					valueSet[v] = true
				}
				merged[key] = make([]string, 0, len(valueSet))
				for v := range valueSet {
					merged[key] = append(merged[key], v)
				}
			}
		}
	}

	addEntries(map1)
	addEntries(map2)

	return merged
}
