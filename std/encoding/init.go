package encoding

var LOCALHOST = NewStringComponent(TypeGenericNameComponent, "localhost")
var LOCALHOP = NewStringComponent(TypeGenericNameComponent, "localhop")

// Initializes component conventions for the NDN library.
func init() {
	initComponentConventions()
}
