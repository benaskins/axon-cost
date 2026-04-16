module github.com/benaskins/axon-cost

go 1.26.1

replace (
	github.com/benaskins/axon-fact => ../axon-fact
	github.com/benaskins/axon-talk => ../axon-talk
)

require (
	github.com/benaskins/axon-fact v0.0.0-00010101000000-000000000000
	github.com/benaskins/axon-talk v0.0.0-00010101000000-000000000000
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/benaskins/axon-tape v0.1.1 // indirect
	github.com/benaskins/axon-tool v0.3.0 // indirect
)
