#!/bin/sh

# 此处备忘
# 根证书（root certificate）为自签名证书，用来签发其他ca证书，
# 也可以直接签发x509证书。
# 由根证书签发的ca可以签发其他x509证书。

# Whatever method you use to generate the certificate and key files,
# the Common Name value used for the server and client certificates/keys
# must each differ from the Common Name value used for the CA certificate.
# Otherwise, the certificate and key files will not work for servers
# compiled using OpenSSL.

ROOT_CA_CN=libra
ROOT_CA_KEY=rootca.key.pem
ROOT_CA_CRT=rootca.crt.pem

# 生成根证书
function generate_root_certificate() {
    # 生成aes256(4096 bits)无密码私钥($ROOT_CA_KEY)
    # 生成x509格式，有效期为366天的根证书($ROOT_CA_CRT)
    openssl req -newkey rsa:4096 -nodes -keyout $ROOT_CA_KEY \
        -new -x509 -days 3660 -sha256 -extensions v3_ca \
        -subj "/C=CN/ST=Shanghai/O=Libra Project/CN=$ROOT_CA_CN" \
        -out $ROOT_CA_CRT
}

# 使用根证书签发一个证书
function generate_certificate() {
    # create private key
    openssl genrsa -out $1.key.pem 2048
    # create request file
    openssl req -new -sha256 -key $1.key.pem \
        -subj "/C=CN/ST=Shanghai/O=Libra Project/CN=$2" -out $1.csr.pem
    # generate the certificate with ca root
    openssl x509 -req -in $1.csr.pem -CA $ROOT_CA_CRT -CAkey $ROOT_CA_KEY \
        -CAcreateserial -out $1.crt.pem -days 366 -sha256
    # verify certificate
    openssl verify -CAfile $ROOT_CA_CRT $1.crt.pem
}

function main() {
    rm -rf *.pem
    generate_root_certificate
    generate_certificate server ibra.net
}

main $*
