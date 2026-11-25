FROM docker.io/library/golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY *.go ./
COPY web ./web
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /taskmanager . \
 && mkdir /data

FROM scratch
COPY --from=build /taskmanager /taskmanager
COPY --from=build /data /data
ENV ADDR=:8080 DATA_FILE=/data/tasks.json
EXPOSE 8080
VOLUME /data
ENTRYPOINT ["/taskmanager"]
