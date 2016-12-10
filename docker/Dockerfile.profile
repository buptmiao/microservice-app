FROM centos
ADD https://github.com/buptmiao/microservice-app/releases/download/v1.0.1/microservice-app-v1.0.1-linux-amd64.tar.gz .
RUN tar -xzf microservice-app-v1.0.1-linux-amd64.tar.gz -C .

EXPOSE 8083 6063
ENTRYPOINT ["./microservice-app-v1.0.1-linux-amd64/profile"]