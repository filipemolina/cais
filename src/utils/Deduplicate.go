package utils

/*
Deduplicates an array of strings.
*/
func Deduplicate(input []string) []string {
	seen := make(map[string]struct{})

	result := make([]string, 0, len(input))

	for _, item := range input {
		if _, exists := seen[item]; !exists {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}

	return result
}
