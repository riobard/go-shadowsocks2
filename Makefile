NAME=shadowsocks2
BINDIR=bin
GOBUILD=CGO_ENABLED=0 go build -ldflags '-w -s'
# The -w and -s flags reduce binary sizes by excluding unnecessary symbols and debug info

all: linux macos win64

linux:
	GOARCH=amd64 GOOS=linux $(GOBUILD) -o $(BINDIR)/$(NAME)-$@

macos:
	GOARCH=amd64 GOOS=darwin $(GOBUILD) -o $(BINDIR)/$(NAME)-$@

win64:
	GOARCH=amd64 GOOS=windows $(GOBUILD) -o $(BINDIR)/$(NAME)-$@.exe

releases: linux macos win64
	gzip $(BINDIR)/shadowsocks2-linux
	gzip $(BINDIR)/shadowsocks2-macos
	zip -m $(BINDIR)/shadowsocks2-win64.zip $(BINDIR)/shadowsocks2-win64.exe

clean:
	rm $(BINDIR)/*