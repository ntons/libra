FROM {{ 'FROM_IMAGE'|env }}

# install packages from apt
RUN set -x && \
    apt-get update && \
    apt-get install -y curl git net-tools python3-jinja2 && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# install getenvoy and retrieve binaries
RUN set -x && \
    curl -sL https://getenvoy.io/cli | bash -s -- -b /usr/local/bin && \
    getenvoy fetch standard:1.14.4

# envoy executor
ENV ENVOY "getenvoy run standard:1.14.4 --"

# work directory
WORKDIR /data

# copy files
COPY envoy          envoy
COPY tools          tools
COPY entrypoint.sh  entrypoint.sh

################################################################################
# Etcd configurations
################################################################################
ENV ETCD_PREFIX     /libra.net
ENV ETCD_ENDPOINTS  ""

################################################################################
## Envoy configurations
################################################################################
# envoy will start using confuguration $envoy_config.
# if envoy_config_template was set, $envoy_config will be generated 
# from $envoy_config_template every time container start.
ENV ENVOY_CONFIG            envoy/envoy.yaml
ENV ENVOY_CONFIG_TEMPLATE   ""

################################################################################
## Node configurations
################################################################################
ENV NODE_ID      ""
ENV NODE_CLUSTER ""
# load balancing weight
ENV NODE_WEIGHT  10

# ingress service options
ENV INGRESS_IFACE eth0
ENV INGRESS_PORT  10000

################################################################################
## Server configurations
################################################################################
# Server context directory
ENV SERVER server

# If multiple SERVER_FROM options are set, select by following declare order
# Get server resources from git
# eg: https://path/to/git/repository@tag
ENV SERVER_FROM_GIT ""

# Get service resources from archive
# supported archive type: zip, gz, bz2
ENV SERVER_FROM_URL ""

ENTRYPOINT [ "./entrypoint.sh" ]

