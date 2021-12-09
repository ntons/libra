#!/bin/bash

# minikube docker env
[ $(which minikube|wc -l) -eq 1 ] && \
    echo -e "\e[1;31mUse minikube docker env\e[0m" && \
    eval $(minikube docker-env)
