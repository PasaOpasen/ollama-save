```sh
make build

cp ollama_save/ollama_save ollama_save
sudo chmod +x ollama_save

./ollama_save --help

./ollama_save save <name1> <name2> <name3:tag> -o models.tar.gz
./ollave_save load models.tar.gz
```
