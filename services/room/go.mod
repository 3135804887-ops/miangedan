module miangedan/services/room

go 1.26

require (
	miangedan/services/project v0.0.0
	miangedan/services/region v0.0.0
)

require gopkg.in/yaml.v3 v3.0.1 // indirect

replace miangedan/services/project => ../project

replace miangedan/services/region => ../region
