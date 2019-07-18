FROM alpine:3.10

COPY ./healthd /usr/local/bin/healthd

ENTRYPOINT [ "/usr/local/bin/healthd" ]