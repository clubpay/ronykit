package mcp

// -- Server metadata --------------------------------------------------------

const serverTitle = "RonyUP MCP Server"

// -- Error messages (format templates) --------------------------------------

const (
	errPathRequired        = "path is required"
	errPathTraversal       = "path must be relative and without traversal: %q"
	errFeatureNameRequired = "feature_name is required"
	errFeatureNameInvalid  = "feature_name must be a valid Go identifier: %q"
)
