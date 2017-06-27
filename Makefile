app:
	go build -v -i -o $@
dev: app
	spk dev

.PHONY: dev app
