package privet

type (
	/*
	Args represents map of arguments that is used for interpolating translated phrase.
	TODO: example
	*/
	Args map[string]interface{}
)

/*

*/
func (a Args) applyTo(phrase string) string {
	return phrase
}
