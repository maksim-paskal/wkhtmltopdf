FROM ubuntu:latest
RUN apt update && apt install -y wkhtmltopdf
COPY ./wkhtmltopdf /wkhtmltopdf

ENV XDG_CACHE_HOME=/tmp/cache
ENV XDG_RUNTIME_DIR=/tmp/runtime
USER 1000

ENTRYPOINT ["/wkhtmltopdf"]