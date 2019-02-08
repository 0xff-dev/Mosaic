FROM golang
ADD . /go/src/Mosaic
WORKDIR /go/src/Mosaic
RUN go install
ENV http_proxy socks5://127.0.0.1:1080
ENV https_proxy socks5://127.0.0.1:1080
ENTRYPOINT /go/bin/Mosaic
EXPOSE 8080
