ARG GO_VERSION=1.9.2
FROM golang:${GO_VERSION}-alpine AS build-stage
RUN apk add --no-cache tzdata
ENV TZ Asia/Taipei
WORKDIR /go/src/gitlab/inu-utility
COPY ./ /go/src/gitlab/inu-utility/
RUN go install

FROM alpine:3.7
COPY --from=build-stage /go/bin/inu-utility /usr/bin/
RUN apk add --no-cache tzdata curl mongodb-tools
ENV TZ Asia/Taipei
ENTRYPOINT ["inu-utility"]
