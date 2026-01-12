module ppo/business

go 1.18

require (
	github.com/google/uuid v1.6.0
	golang.org/x/crypto v0.23.0
	ppo/data v0.0.0
	ppo/sdk v0.0.0
)

require github.com/lib/pq v1.10.9 // indirect

replace ppo/data => ../data

replace ppo/sdk => ../../sdk
