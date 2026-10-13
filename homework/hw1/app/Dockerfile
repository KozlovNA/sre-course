FROM golang:1.26-alpine
ARG VERSION
ENV VERSION=$VERSION
WORKDIR /app
COPY . .
RUN go build -o server .
CMD ["./server"]
