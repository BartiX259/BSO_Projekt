all:
	docker build -t bso-docker .
	docker run -p 8080:8080 --rm -v $PWD:/app -v /app/tmp --name bso-docker-air bso-docker