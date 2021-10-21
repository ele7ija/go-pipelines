# Pokretanje
### 1. Baza podataka

```bash
docker run --name go-pipelines-postgres -e POSTGRES_PASSWORD=go-pipelines -e POSTGRES_USER=go-pipelines -e POSTGRES_DB=go-pipelines -d -p 5432:5432 postgres
&&
docker exec -it go-pipelines-postgres bash
&&
psql -U go-pipelines
```

```sql
CREATE TABLE image (id serial PRIMARY KEY, name VARCHAR, fullpath VARCHAR, thumbnailpath VARCHAR, resolution_x INT, resolution_y INT);
CREATE TABLE "user" (id serial PRIMARY KEY, username VARCHAR, password VARCHAR); # sifra bojan
CREATE TABLE user_images (user_id INT NOT NULL, image_id INT NOT NULL, PRIMARY KEY (user_id, image_id), FOREIGN KEY (user_id) REFERENCES "user"(id), FOREIGN KEY (image_id) REFERENCES image(id));
```

### 2. Prikupljanje podataka o performansama (opciono)

```bash
docker run --name influxdb -e DOCKER_INFLUXDB_INIT_USERNAME=go-pipelines -e DOCKER_INFLUXDB_INIT_PASSWORD=go-pipelines -e DOCKER_INFLUXDB_INIT_ORG=go-pipelines -e DOCKER_INFLUXDB_INIT_BUCKET=stats -d -p 8086:8086 influxdb
&&
docker run -d -p 9090:3000/tcp --link influxdb --name=grafana grafana/grafana:4.1.0
```
Otvoriti u browseru `localhost:9090`

### 3. Pokrenuti web server

```bash
go run cmd/go-pipelines/main.go
```

Otvoriti u browseru `localhost:3333`. 

Napomena: nije potrebno zasebno pokretati frontend server, frontend je buildovan i stavljen 
u *static* već.
