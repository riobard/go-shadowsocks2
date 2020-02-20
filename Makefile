NAME=shadowsocks2
BINDIR=bin
GOBUILD=CGO_ENABLED=0 go build -ldflags '-w -s'
# The -w and -s flags reduce binary sizes by excluding unnecessary symbols and debug info

all: linux macos win64 win32

linux:
	GOARCH=amd64 GOOS=linux $(GOBUILD) -o $(BINDIR)/$(NAME)-$@

macos:
	GOARCH=amd64 GOOS=darwin $(GOBUILD) -o $(BINDIR)/$(NAME)-$@

win64:
	GOARCH=amd64 GOOS=windows $(GOBUILD) -o $(BINDIR)/$(NAME)-$@.exe

win32:
	GOARCH=386 GOOS=windows $(GOBUILD) -o $(BINDIR)/$(NAME)-$@.exe


test: test-linux test-macos test-win64 test-win32

test-linux:
	GOARCH=amd64 GOOS=linux go test

test-macos:
	GOARCH=amd64 GOOS=darwin go test

test-win64:
	GOARCH=amd64 GOOS=windows go test

test-win32:
	GOARCH=386 GOOS=windows go test

releases: linux macos win64 win32
	chmod +x $(BINDIR)/$(NAME)-*
	gzip $(BINDIR)/$(NAME)-linux
	gzip $(BINDIR)/$(NAME)-macos
	zip -m -j $(BINDIR)/$(NAME)-win32.zip $(BINDIR)/$(NAME)-win32.exe
	zip -m -j $(BINDIR)/$(NAME)-win64.zip $(BINDIR)/$(NAME)-win64.exe

clean:
	rm $(BINDIR)/*

# Remove trailing {} from the release upload url
GITHUB_UPLOAD_URL=$(shell echo $${GITHUB_RELEASE_UPLOAD_URL%\{*})

upload: releases
	curl -H "Authorization: token $(GITHUB_TOKEN)" -H "Content-Type: application/gzip" --data-binary @$(BINDIR)/$(NAME)-linux.gz  "$(GITHUB_UPLOAD_URL)?name=$(NAME)-linux.gz"
	curl -H "Authorization: token $(GITHUB_TOKEN)" -H "Content-Type: application/gzip" --data-binary @$(BINDIR)/$(NAME)-macos.gz  "$(GITHUB_UPLOAD_URL)?name=$(NAME)-macos.gz"
	curl -H "Authorization: token $(GITHUB_TOKEN)" -H "Content-Type: application/zip"  --data-binary @$(BINDIR)/$(NAME)-win64.zip "$(GITHUB_UPLOAD_URL)?name=$(NAME)-win64.zip"
	curl -H "Authorization: token $(GITHUB_TOKEN)" -H "Content-Type: application/zip"  --data-binary @$(BINDIR)/$(NAME)-win32.zip "$(GITHUB_UPLOAD_URL)?name=$(NAME)-win32.zip"