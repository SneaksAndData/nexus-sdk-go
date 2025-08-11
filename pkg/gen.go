package pkg

//go:generate go run github.com/ogen-go/ogen/cmd/ogen --target generated/scheduler -package api --clean /Users/gzu/GolandProjects/nexus/docs/openapi.json

//go:generate go run github.com/ogen-go/ogen/cmd/ogen --target generated/receiver -package api --clean /Users/gzu/GolandProjects/nexus-receiver/docs/openapi.json
