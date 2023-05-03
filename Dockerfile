FROM golang:latest AS build

WORKDIR $GOPATH/src/app
COPY . .

RUN CGO_ENABLED=0 go build -o /app

FROM gcr.io/distroless/static:latest
COPY --from=build /app /app

CMD ["/app"]
