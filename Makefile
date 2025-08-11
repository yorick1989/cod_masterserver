build:
	@CGO_ENABLED=0 go build -ldflags "-s -w" -o bin/master cmd/gsmaster/*.go

compress:
	@[ -x /usr/bin/upx ] && upx --best --lzma bin/master

run:
	@CGO_ENABLED=0 go run cmd/gsmaster/*.go

all: build compress
