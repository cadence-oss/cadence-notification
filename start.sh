#!/bin/sh

set -ex

CONFIG_TEMPLATE_PATH="${CONFIG_TEMPLATE_PATH:-/etc/cadence-notification/config/config_template.yaml}"

dockerize -template $CONFIG_TEMPLATE_PATH:/etc/cadence-notification/config/docker.yaml

# TODO https://github.com/cadence-oss/cadence-notification/issues/23
# depends_on in docker-compose doesn't work and the startup will fail at waiting for Kafka to be up.
# waiting for 5 second here can mitigate it.
sleep 5
exec cadence-notification --root $CADENCE_NOTIFICATION_HOME --env docker start --services=$SERVICES

