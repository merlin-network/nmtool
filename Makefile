install:
	go install -ldflags "-X github.com/fanfury-sports/nmtool/config/generate.ConfigTemplatesDir=$(CURDIR)/config/templates"

generate-nemo-genesis:
	bash ./config/generate/genesis/generate-nemo-genesis.sh

# when keys are added or changed, use me. we don'd replace keys by default because they include
# creation time, so they create noise by always creating a diff.
generate-nemo-genesis-with-keys:
	REPLACE_ACCOUNT_KEYS=true bash ./config/generate/genesis/generate-nemo-genesis.sh

generate-ibc-genesis:
	CHAIN_ID=highbury_710-2 DEST=./config/templates/ibcchain/master/initstate/.nemo DENOM=uatom SKIP_INCENTIVES=true bash ./config/generate/genesis/generate-nemo-genesis.sh
