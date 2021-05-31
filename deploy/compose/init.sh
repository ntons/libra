#!/bin/sh

# insert greeter app into mongodb

docker-compose exec mongo mongo librad_config --eval 'db.apps.insert({"_id":"greeter","key":10000001,"secret":"3f67ae95ed060e33d5ac351db031f1c6","fingerprint":"7ec289b5ad1edb1c61a19f1a0945d8e5","permissions":["helloworld.Greeter"]});'
docker-compose exec mongo mongo librad_config --eval 'db.apps.find({"_id":"greeter"})'
