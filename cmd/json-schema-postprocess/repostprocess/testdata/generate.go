package testdata

import (
	"embed"
)

//go:generate go tool go-jsonschema ./color/color.schema.json -o=./color/model.gen.go -p=color
//go:generate go tool go-jsonschema ./confusing/confusing.schema.json -o=./confusing/model.gen.go -p=confusing
//go:generate go tool go-jsonschema ./simple/simple.schema.json -o=./simple/model.gen.go -p=simple

//go:embed *
var TestData embed.FS

func GetTestData(path string) ([]byte, error) {
	return TestData.ReadFile(path)
}
