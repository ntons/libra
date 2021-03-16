ARG BASE

FROM $BASE

COPY librad      /bin/
COPY librad.yaml /etc/

ENTRYPOINT ["/bin/librad","-c","/etc/librad.yaml"]

