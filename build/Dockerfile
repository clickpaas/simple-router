FROM golang:1.19.4 AS base

ENV GOPROXY "https://goproxy.cn"

RUN mkdir  /simple-router

ADD . /simple-router/


RUN cd /simple-router/cmd/simplerouter && go build 



FROM alpine

COPY --from=base /simple-router/cmd/simplerouter/simplerouter /opt/simplerouter

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories && apk add tzdata && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && echo "Asia/Shanghai" > /etc/timezone && apk add libc6-compat && chmod +x /opt/simplerouter


ADD ./build/white.txt /opt/white.txt

ENTRYPOINT ["/opt/simplerouter"]
