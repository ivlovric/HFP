all:
	go build -ldflags "-s -w"  -o HFP *.go

debug:
	go build -o HFP *.go

.PHONY: clean
clean:
	rm -fr HFP
