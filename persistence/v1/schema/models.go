package schema

import _ "embed"

//go:embed sql/create.sql
var schema string

//go:embed sql/drop.sql
var dropSchema string
