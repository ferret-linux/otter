package config

import "reflect"

// MergeChanged copies only the fields that an interactive edit actually
// changed from original onto current. It lets a settings editor save its own
// edits under a config lock without reverting values that another process
// committed while the editor was open.
func MergeChanged(original, edited, current *FileConfig) *FileConfig {
	out := *current
	mergeChangedValue(reflect.ValueOf(&out).Elem(), reflect.ValueOf(original).Elem(), reflect.ValueOf(edited).Elem())
	return &out
}

func mergeChangedValue(dst, original, edited reflect.Value) {
	if dst.Kind() != reflect.Struct {
		if dst.CanSet() && !reflect.DeepEqual(original.Interface(), edited.Interface()) {
			dst.Set(edited)
		}
		return
	}
	for i := range dst.NumField() {
		mergeChangedValue(dst.Field(i), original.Field(i), edited.Field(i))
	}
}
