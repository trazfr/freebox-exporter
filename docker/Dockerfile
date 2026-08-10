FROM golang
WORKDIR /app
RUN go install github.com/trazfr/freebox-exporter@latest

COPY docker/scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT [ "/entrypoint.sh" ]
CMD [ "-debug" ]