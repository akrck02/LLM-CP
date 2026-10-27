package main

// wordPieces cuts a word longer than a line (a path, a link) into pieces of at most most letters,
// so a bubble never runs past its edge; a shorter word comes back whole.
func wordPieces(word string, most int) []string {
	letters := []rune(word)
	if most < 1 || len(letters) <= most {
		return []string{word}
	}
	pieces := make([]string, 0, len(letters)/most+1)
	for len(letters) > most {
		pieces = append(pieces, string(letters[:most]))
		letters = letters[most:]
	}
	return append(pieces, string(letters))
}
