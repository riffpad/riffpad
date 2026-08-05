FROM golang:1.25-alpine AS build
WORKDIR /src
COPY apps/relay/go.mod apps/relay/go.sum ./apps/relay/
COPY packages/protocol ./packages/protocol/
COPY packages/webui ./packages/webui/
COPY apps/relay ./apps/relay/
RUN cd apps/relay && CGO_ENABLED=0 go build -o /out/relay .

FROM alpine:3.20
RUN adduser -D -H riffpad
COPY --from=build /out/relay /usr/local/bin/relay
USER riffpad
EXPOSE 9090
ENTRYPOINT ["relay"]
