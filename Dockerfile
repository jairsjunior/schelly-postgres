FROM golang:1.12.3 AS BUILD

RUN mkdir /schelly-postgres
WORKDIR /schelly-postgres

ADD go.mod .
ADD go.sum .
RUN go mod download

#now build source code
ADD schelly-postgres/ ./

# RUN go test -v postgresprovider_test.go postgresprovider.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o /go/bin/schelly-postgres .


FROM postgres:10

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get -y --no-install-recommends install ca-certificates curl
#  && rm -rf /var/cache/apk/*
EXPOSE 7070

# ENV RESTIC_PASSWORD ''
ENV LISTEN_PORT 7070
ENV LISTEN_IP '0.0.0.0'
ENV LOG_LEVEL 'debug'

ENV TARGET_DATA_BACKEND 'file'

ENV SIMULTANEOUS_WRITES '3'
ENV MAX_BANDWIDTH_WRITE '0'
ENV SIMULTANEOUS_READS '10'
ENV MAX_BANDWIDTH_READ '0'

ENV PRE_POST_TIMEOUT '7200'
ENV PRE_BACKUP_COMMAND ''
ENV POST_BACKUP_COMMAND ''

COPY --from=BUILD /go/bin/* /bin/
ADD startup.sh /

CMD [ "/startup.sh" ]
