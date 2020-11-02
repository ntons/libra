#!/bin/sh

# 自签名证书

O="Libra Project"
CN="libra.io"

openssl req -x509 -sha256 -nodes -days 365 -newkey rsa:2048 -subj "/O=$O/CN=$CN" -keyout $CN.key -out $CN.crt
#openssl req -out $CN.csr -newkey rsa:2048 -nodes -keyout $CN.key -subj "/O=$O/CN=$CN"
#openssl x509 -req -days 365 -CA $CN.crt -CAkey $CN.key -set_serial 0 -in CN.csr -out $CN.com.crt

