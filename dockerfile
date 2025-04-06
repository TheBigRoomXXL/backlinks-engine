# !!! NOT A PRODUCTION IMAGE !!!
# This image is only meant to be used during developement to avoid issue with duckcb on alpine

# Build: docker build -t be . 
# Run: docker run -v /home/sebastien/.cache/go-build/:/home/dev/.cache/go-build/ be

FROM golang:1.24 AS builder

ARG CPU_ARCH=amd64
ARG DUCKDB_VERSION=v1.2.1

RUN apt-get update; \
  apt-get -y install unzip


RUN wget -nv https://github.com/duckdb/duckdb/releases/download/${DUCKDB_VERSION}/libduckdb-linux-${CPU_ARCH}.zip -O libduckdb.zip;
RUN unzip libduckdb.zip -d /tmp/libduckdb
RUN mv /tmp/libduckdb/libduckdb.so /usr/lib/

RUN useradd -ms /bin/bash dev
USER dev
WORKDIR /home/dev/be/

COPY go.* .
RUN go mod download


ENV LD_LIBRARY_PATH=/usr/lib/
ENV CGO_ENABLED=1
ENV CGO_LDFLAGS="-L/tmp/libduckdb"
ENV GOCACHE="/home/dev/be/.cache"
ENTRYPOINT ["go", "run" ,"-tags=duckdb_use_lib", "."]


