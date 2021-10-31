# Running the project on your local machine

## 1. Running via docker compose

```bash
$ docker compose up -d
```

Visit `localhost:3333` for the website 
or `localhost:9090` for Grafana-visualized performance data.

## 2. Running semi-manually (archived)
### 1. Database

```bash
docker run --name go-pipelines-postgres -e POSTGRES_PASSWORD=go-pipelines -e POSTGRES_USER=go-pipelines -e POSTGRES_DB=go-pipelines -d -p 5432:5432 postgres
&&
docker exec -it go-pipelines-postgres bash
&&
psql -U go-pipelines
```
And run the SQL script from `.dockerdb/init.sql`

### 2. Performance statistics aggregation (optional)

```bash
docker run --name influxdb -d -p 8086:8086 influxdb:1.7
&&
docker run -d -p 9090:3000/tcp --link influxdb --name=grafana grafana/grafana:4.1.0
```
Open `localhost:9090` in a web browser.

### 3. Running the webserver

```bash
go run cmd/go-pipelines/main.go
```

Open `localhost:3333` in a web browser.

Info: frontend is already deployed in `/static`
