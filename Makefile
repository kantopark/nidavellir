db:
	docker-compose up -d

stop:
	docker-compose down

clear: stop
	docker volume prune -f
	rm -f nida.exe

build:
	go build -o nida.exe .
