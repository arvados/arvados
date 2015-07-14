/* Ensures that we only have one copy of each unique string. This is
/* not designed for concurrent access. */
package summary

// This code should probably be moved somewhere more universal.

type CanonicalString struct {
	m map[string]string
}

func (cs *CanonicalString) Get(s string) (r string) {
	if cs.m == nil {
		cs.m = make(map[string]string)
	}
	value, found := cs.m[s]
	if found {
		return value
	}

	// s may be a substring of a much larger string.
	// If we store s, it will prevent that larger string from getting
	// garbage collected.
	// If this is something you worry about you should change this code
	// to make an explict copy of s using a byte array.
	cs.m[s] = s
	return s
}
