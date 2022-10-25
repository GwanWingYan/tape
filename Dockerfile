FROM golang:alpine as golang
LABEL stage=builder
WORKDIR /root
RUN apk add --no-cache git
ENV GOPROXY=https://goproxy.cn,direct
ENV export GOSUMDB=off
COPY . .

FROM golang as build
LABEL stage=builder
ARG GITHUB_USER
ARG GITHUB_PAT
RUN go env -w GOPRIVATE=github.com/GwanWingYan &&\
  git config --global --add url."https://${GITHUB_USER}:${GITHUB_PAT}@github.com/".insteadOf "https://github.com/" &&\
  # go build -o tape_benchmark ./benchmark &&\
  go build -o tape ./cmd/tape

FROM alpine
RUN mkdir -p /config
# COPY --from=build /root/tape_benchmark /usr/local/bin
COPY --from=build /root/tape /usr/local/bin
CMD ["tape"]
