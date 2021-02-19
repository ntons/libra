# kubernetes deplyment scripts

## Profiles

Setup the infrastructures and apply librad's configuration to configmap.

* Profile should be applied before librad.

### Development

Run infrastructures in cluster, and configure librad to use them.

### Production

Infrastructures should be run outside of cluster, configure librad to access them manually.

## Configurations

Configuration files.
 
* Configuration should be applied before librad.

## Manifests

Manifests for librad which implement libra api

* Each service was deployed independently, so that updates to service could be applied minimally

