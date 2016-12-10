FROM centos
ADD https://github.com/buptmiao/microservice-app/releases/download/v1.0.1/microservice-app-v1.0.1-linux-amd64.tar.gz .
RUN tar -xzf microservice-app-v1.0.1-linux-amd64.tar.gz -C .

EXPOSE 8082 6062
ENTRYPOINT ["./microservice-app-v1.0.1-linux-amd64/feed"]