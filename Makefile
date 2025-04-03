docker:
	docker build -t bso-docker .
	docker run -p 8080:8080 bso-docker
