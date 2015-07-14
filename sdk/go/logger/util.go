// Helper methods for interacting with Logger.
package logger

// Retrieves the map[string]interface{} stored at parent[key] if it
// exists, otherwise it makes it and stores it there.
// This is useful for logger because you may not know if a map you
// need has already been created.
func GetOrCreateMap(
	parent map[string]interface{},
	key string) (child map[string]interface{}) {
	read, exists := parent[key]
	if exists {
		child = read.(map[string]interface{})

	} else {
		child = make(map[string]interface{})
		parent[key] = child
	}
	return
}
