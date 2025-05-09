FROM alpine:latest AS stage1

RUN apk add -U --no-cache ca-certificates

RUN apk add bash curl && \
  curl -fSsL https://raw.githubusercontent.com/vishnubob/wait-for-it/master/wait-for-it.sh -o /wait-for-it.sh && \
  chmod +x /wait-for-it.sh && \
  apk del curl

FROM golang:1.24 AS build

#!!!DO NOT REMOVE THE BELOW 3 LINES!!! Needed for github action to work
#GHAONLY:COPY dotssh/* /root/.ssh/
#GHAONLY:COPY dotgitconfig /root/
#GHAONLY:RUN cat /root/dotgitconfig >> /root/.gitconfig

# Download public key for github.com
RUN mkdir -p -m 0600 ~/.ssh && ssh-keyscan github.com >> ~/.ssh/known_hosts

# Use git protocol whenever possible
RUN echo '[url "git@github.com:"]\n\tinsteadOf = https://github.com/\n' >> ~/.gitconfig

WORKDIR /build

# Put the go modules in own layer to speed up build
COPY go.mod .
COPY go.sum .
RUN --mount=type=ssh go mod download

# Add remaining code
COPY . .

ARG location=github.com/fluffy-bunny/fluffycore/example/server
ARG commit=AFFE
ARG version=0.0.0
ARG date=19700101
ARG BINARY=server
ENV BINARY=${BINARY}

ENV GOOS=linux
ENV GOARCH=${TARGETARCH}
ENV CGO_ENABLED=0

ENV EXTLDFLAGS="-static"
ENV LDFLAGS="-X ${location}.date=${date} -X ${location}.version=${version} -X ${location}.commit=${commit} -s -w -extldflags '${EXTLDFLAGS}'"
RUN --mount=type=ssh go build -a -tags netgo -ldflags "$LDFLAGS" ./example/$BINARY

FROM alpine:latest

LABEL maintainer="ghstahl@gmail.com"
LABEL description="fluffycore example"

ARG BINARY=server
ENV BINARY=${BINARY}

WORKDIR /app
COPY --from=build /build/${BINARY} ./

ENV PATH=/app:${PATH}


ENTRYPOINT ["/app/server"]