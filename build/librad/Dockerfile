FROM ubuntu:20.04

#RUN set -x && \
#        apt-get update && \
#        apt-get install -y curl dnsutils net-tools && \
#        apt-get clean && \
#        rm -rf /var/lib/apt/lists/*

COPY entrypoint.sh  /
COPY librad         /bin/
COPY librad.yaml    /etc/

ENTRYPOINT ["./entrypoint.sh"]

CMD ["/bin/librad","-c","/etc/librad.yaml"]

