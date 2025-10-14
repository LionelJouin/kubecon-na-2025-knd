FROM golang:1.25 AS build

ENV GO111MODULE=on

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-extldflags -static" -o kubecon-na-2025-knd ./cmd

FROM alpine:3.22

COPY --from=build /app/kubecon-na-2025-knd .

CMD ["./kubecon-na-2025-knd"]