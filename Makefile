OAPI_GEN := $(shell oapi-codegen -version 2> /dev/null)

generate-client:
ifndef OAPI_GEN
	go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.10.1
endif
	oapi-codegen -generate types,client -package client .openapi/definition.yml > client/client.go