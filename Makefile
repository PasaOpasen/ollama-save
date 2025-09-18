
load-script:
	rm -rf ollama_save/util
	mkdir -p ollama_save/util
	wget https://gist.github.com/Mazyod/3bfcb4ec1aaa9b61a877d8ba1a308624/raw/81a25de468dbb891eccd3b1a4dbb8e94def155ca/ollamautil.go -O ollama_save/util/ollamautil.go


build:
	cd ollama_save; go build -buildvcs=false