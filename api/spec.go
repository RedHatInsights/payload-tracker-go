// Package api embeds the service's OpenAPI contract so it can be served
// directly by the running binary.
package api

import _ "embed"

//go:embed api.spec.json
var Spec []byte
