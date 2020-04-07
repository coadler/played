FROM golang:1.13.7

ENV FDB_URL "https://www.foundationdb.org/downloads/6.2.15/ubuntu/installers/foundationdb-clients_6.2.15-1_amd64.deb"
RUN apt update && apt install -y wget zlib1g zlib1g-dev
RUN wget -O fdb.deb $FDB_URL &&  dpkg -i fdb.deb

COPY . /go/src/github.com/coadler/played
ENV GO111MODULE=on

RUN cd /go/src/github.com/coadler/played/cmd/playedd && go build -o /go/playedd .
	
FROM ubuntu:18.04

ENV FDB_URL "https://www.foundationdb.org/downloads/6.2.15/ubuntu/installers/foundationdb-clients_6.2.15-1_amd64.deb"
RUN apt update && apt install -y wget zlib1g zlib1g-dev
RUN wget -O fdb.deb $FDB_URL &&  dpkg -i fdb.deb

COPY --from=0 /go/playedd /
COPY entrypoint.sh /
CMD [ "/entrypoint.sh" ]
