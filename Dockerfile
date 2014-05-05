# BUILD-USING:        docker build .
# RUN-USING:          docker run -p 8080:8080

FROM ubuntu

MAINTAINER github.com/boourns

#RUN echo 'deb http://archive.ubuntu.com/ubuntu precise main universe' > /etc/apt/sources.list && \
#    echo 'deb http://archive.ubuntu.com/ubuntu precise-updates universe' >> /etc/apt/sources.list && \
#    apt-get update

# Utilities
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y build-essential less curl git wget

RUN mkdir -p /usr/local/src

# Go 1.2
RUN cd /usr/local/src && \
    wget https://go.googlecode.com/files/go1.2.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.2.linux-amd64.tar.gz

ENV PATH $PATH:/usr/local/go/bin

# Initialize environment variables.
ENV GOPATH /go
ENV GOBIN /go/bin
ENV ROOT_PATH /go/src/github.com/boourns/

# Set up required directories.
RUN mkdir -p $GOBIN
RUN mkdir -p $ROOT_PATH

# Download Incredible to its appropriate GOPATH location.
RUN cd $ROOT_PATH && \
    wget -O incredible.tar.gz https://github.com/boourns/incredible/archive/master.tar.gz && \
    tar zxvf incredible.tar.gz && \
    mv incredible-master incredible && \
    cd incredible

# Retrieve Sky dependencies.
RUN cd $ROOT_PATH/incredible && go get

# Build and install skyd into GOBIN.
RUN cd $ROOT_PATH/incredible && go build

ENTRYPOINT $ROOT_PATH/incredible/start.sh

EXPOSE 8080

