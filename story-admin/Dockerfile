FROM golang:1.25-alpine AS build
WORKDIR /src
COPY main.go ./
RUN CGO_ENABLED=0 go build -o /story-admin main.go

FROM scratch
COPY --from=build /story-admin /story-admin
EXPOSE 8081
ENTRYPOINT ["/story-admin"]
