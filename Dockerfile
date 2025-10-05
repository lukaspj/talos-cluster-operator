FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build

RUN apk update && apk upgrade && apk add --no-cache ca-certificates
RUN update-ca-certificates

ARG SHA
ARG DATE

COPY . /src
WORKDIR /src

ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags "-X cmd.commit=$SHA -X cmd.date=$DATE" -o talos-cluster-operator main.go

FROM scratch AS runtime

COPY --from=build /src/talos-cluster-operator /talos-cluster-operator
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/talos-cluster-operator", "operator"]