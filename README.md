# BSO Projekt

## Uruchomienie w docker

```
docker build -t bso-docker .
docker run -p 8080:8080 bso-docker
```
lub
```
make docker
```

## Development

1. Zainstalować [go](https://go.dev/doc/install)
2. Zainstalować [air](https://github.com/air-verse/air)
```
go install github.com/air-verse/air@latest
```
3. Uruchomić `air`
4. Wejść na http://localhost:8080

Zmiany w plikach źródłowych powinny być wykrywane przez `air`, wystarczy odświeżyć kartę w przeglądarce.
