FROM ubuntu:18.04

RUN apt update && apt upgrade -y \
 && apt install -y git wget

ENV GOLANG_VERSION 1.12
ENV goRelArch linux-amd64

RUN wget https://golang.org/dl/go${GOLANG_VERSION}.${goRelArch}.tar.gz \
 && tar -C /usr/local -xzf go${GOLANG_VERSION}.${goRelArch}.tar.gz \
 && rm go${GOLANG_VERSION}.${goRelArch}.tar.gz

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

RUN go get google.golang.org/api/calendar/v3 \
 && go get golang.org/x/oauth2/google \
 && go get github.com/mirror520/cal \
 && go install github.com/mirror520/cal

WORKDIR $GOPATH/src/github.com/mirror520/cal

CMD cal