package language

func GetCodeFileName(lang string) string {
	switch lang {
	case "C":
		return "main.c"
	case "Cpp":
		return "main.cpp"
	default:
		return "main.c"
	}
}
