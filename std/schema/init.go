package schema

// Initializes registries for node and policy implementations by creating empty maps and populating them with base node, express point, leaf node, and policy descriptors.
func init() {
	NodeRegister = make(map[string]*NodeImplDesc)
	PolicyRegister = make(map[string]*PolicyImplDesc)
	initBaseNodeImplDesc()
	initExpressPointDesc()
	initLeafNodeDesc()
	initPolicies()
}
