module go.einride.tech/mage-tools/.mage

go 1.17

require (
	github.com/magefile/mage v1.12.1
	go.einride.tech/mage-tools v0.0.0-00010101000000-000000000000
)

require (
	github.com/go-logr/logr v1.2.2 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/iancoleman/strcase v0.2.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace go.einride.tech/mage-tools => ../
