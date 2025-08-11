FROM golang:alpine AS builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w" \
    -o /app/bin/master \
    /app/cmd/gsmaster/*.go

RUN chmod +x /app/bin/master

FROM scratch

LABEL maintainer="Yorick Gruijthuijzen <yorick-1989@hotmail.com>" \
      org.opencontainers.image.authors="Yorick Gruijthuijzen <yorick-1989@hotmail.com>"  \
      org.opencontainers.image.title="Call of Duty master server" \
      org.opencontainers.image.description="Call of Duty master server" \
      org.opencontainers.image.licenses="GPL-3.0 license" \
      org.opencontainers.image.url="https://github.com/yorick1989/cod_masterserver" \
      org.opencontainers.image.source="https://github.com/yorick1989/cod_masterserver" \
      org.opencontainers.image.documentation="https://github.com/yorick1989/cod_masterserver"

WORKDIR /src/bin/

COPY --from=builder \
     /etc/ssl/certs/ca-certificates.crt \
     /etc/ssl/certs/

COPY \
  --chmod=111 \
  --from=builder \
  /app/bin/master /

EXPOSE 8080
EXPOSE 20700
EXPOSE 20710

ENTRYPOINT ["/master"]
