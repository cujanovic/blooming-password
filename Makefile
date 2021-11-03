PROGRAM = bin/blooming-password-server
PROGRAM_SOURCE = src/server/blooming-password-server.go
LOAD = tools/blooming-password-filter-create
LOAD_SOURCE = src/filter-create/blooming-password-filter-create.go
REMOTE_DOWN_PASS_URL = https://downloads.pwnedpasswords.com/passwords
REMOTE_PASS_FILENAME = pwned-passwords-sha1-ordered-by-count-v6
DATA_FOLDER = data

.PHONY: build
build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static -s -w"' -o $(PROGRAM) $(PROGRAM_SOURCE)
	GOOS=linux GOARCHC=amd64 GO_ENABLED=0 go build -a -ldflags '-extldflags "-static -s -w"' -o $(LOAD) $(LOAD_SOURCE)

.PHONY: go-optimize
go-optimize:
	strip $(PROGRAM)
	strip $(LOAD)
	upx --brute $(PROGRAM)
	upx --brute $(LOAD)

.PHONY: clean
clean:
	rm -f $(PROGRAM)
	rm -f $(LOAD)

.PHONY: go-deps
go-deps:
	go mod vendor

.PHONY: build-deps-filter
build-deps-filter:
	apt-get update -y
	apt-get install -y make wget p7zip-full coreutils

.PHONY: build-deps-go
build-deps-go:
	apt-get update -y
	apt-get install -y binutils upx-ucl

.PHONY: filter
filter:
	mkdir -p $(DATA_FOLDER)
	date && wget -c $(REMOTE_DOWN_PASS_URL)/$(REMOTE_PASS_FILENAME).7z -O $(DATA_FOLDER)/$(REMOTE_PASS_FILENAME).7z
	date && LC_ALL=C 7z e $(DATA_FOLDER)/$(REMOTE_PASS_FILENAME).7z -aos -o$(DATA_FOLDER)/
	date && LC_ALL=C cut -c 1-16 $(DATA_FOLDER)/$(REMOTE_PASS_FILENAME).txt > $(DATA_FOLDER)/1-16-$(REMOTE_PASS_FILENAME).txt
	date && $(LOAD) $(DATA_FOLDER)/1-16-$(REMOTE_PASS_FILENAME).txt $(DATA_FOLDER)/1-16-$(REMOTE_PASS_FILENAME).filter
	rm -rf $(DATA_FOLDER)/1-16-$(REMOTE_PASS_FILENAME).txt
	rm -rf $(DATA_FOLDER)/$(REMOTE_PASS_FILENAME).txt
	rm -rf $(DATA_FOLDER)/$(REMOTE_PASS_FILENAME).7z

.PHONY: build-go
build-go: clean build-deps-go go-deps build go-optimize

.PHONY: build-filter
build-filter: build-deps-filter filter

.PHONY: all
all: clean build-deps-go go-deps build go-optimize build-deps-filter filter
