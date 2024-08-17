package bot

func Split(r rune) bool {
	return r == ' ' || r == '\n' || r == ','
}
