FROM ubuntu:22.04
ARG BUILDARCH

ADD https://github.com/wkhtmltopdf/packaging/releases/download/0.12.6.1-2/wkhtmltox_0.12.6.1-2.jammy_${BUILDARCH}.deb /tmp/wkhtmltox.deb
RUN apt-get update \
&& apt full-upgrade -y \
&& apt-get install -y /tmp/wkhtmltox.deb \
&& rm /tmp/wkhtmltox.deb

COPY ./wkhtmltopdf /wkhtmltopdf

ENV XDG_CACHE_HOME=/tmp/cache
ENV XDG_RUNTIME_DIR=/tmp/runtime
USER 1000

ENTRYPOINT ["/wkhtmltopdf"]