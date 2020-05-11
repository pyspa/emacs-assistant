GO ?= go
EMACS ?= emacs

libpyspaemacs.so: main.go
	$(GO) build -buildmode=c-shared -ldflags -s -o $@ $<

clean:
	rm -f libpyspaemacs.so

all: libpyspaemacs.so

test: libpyspaemacs.so
	$(EMACS) --batch --load libpyspaemacs.so --eval '(when (featurep (quote pyspa)) (pyspa/echo "ぁっぉ〜"))'
	$(EMACS) --batch --load libpyspaemacs.so --eval '(when (featurep (quote pyspa)) (pyspa/slack-init))'
