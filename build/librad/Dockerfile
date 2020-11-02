FROM ubuntu:18.04

RUN set -x && \
        apt-get update && \
        apt-get install -y curl dnsutils net-tools && \
        apt-get clean && \
        rm -rf /var/lib/apt/lists/*

WORKDIR /data

COPY entrypoint.sh librad librad.yaml ./

ENTRYPOINT [ "./entrypoint.sh" ]

CMD ["./librad", "-c", "./librad.yaml"]

