all: create_registry add_images

create_registry:
	docker run -d -p 5000:5000 --restart=always --name registry registry:2
	echo "created a docker registry"

add_images: build_db_image build_go_image push_grafana push_influx
	echo "all images ready"

push_influx:
	echo "pushing InfluxDB image..."
	docker pull influxdb:1.7
	docker tag influxdb:1.7 localhost:5000/go-pipelines-influx
	docker push localhost:5000/go-pipelines-influx
	echo "...pushed InfluxDB image"

push_grafana:
	echo "pushing Grafana image..."
	docker pull grafana/grafana:4.1.0
	docker tag grafana/grafana:4.1.0 localhost:5000/go-pipelines-grafana
	docker push localhost:5000/go-pipelines-grafana
	echo "...done pushing Grafana image"

build_db_image:
	echo "building DB image..."
	docker build -t localhost:5000/go-pipelines-db .dockerdb
	docker push localhost:5000/go-pipelines-db
	echo "...done building DB image"

build_go_image:
	echo "building Go image..."
	docker build -t localhost:5000/go-pipelines .
	docker push localhost:5000/go-pipelines
	echo "...done building Go image"

clean:
	echo "deleting the registry"
	docker rm --force registry
	echo "deleting all images"
	docker image rm 							\
			localhost:5000/go-pipelines-db 		\
			localhost:5000/go-pipelines 		\
			localhost:5000/go-pipelines-grafana	\
			localhost:5000/go-pipelines-influx

