FROM golang as build-stage
WORKDIR /app

COPY go.mod /app
COPY go.sum /app
RUN go mod download

COPY main.go /app

RUN cd /app && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mqtt-producer

FROM alpine
COPY --from=build-stage /app/mqtt-producer /
CMD ["/mqtt-producer"]