FROM alpine

COPY promruval /usr/bin/promruval
COPY Dockerfile /

ENTRYPOINT ["promruval"]
CMD ["--help"]

