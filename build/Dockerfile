FROM debian:buster

COPY bin /bin
COPY etc /etc

ENTRYPOINT ["/bin/librad","-c","/etc/librad.yaml"]

