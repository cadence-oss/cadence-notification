#!/bin/sh

set -ex

CONFIG_TEMPLATE_PATH="${CONFIG_TEMPLATE_PATH:-/etc/cadence-notification/config/config_template.yaml}"

dockerize -template $CONFIG_TEMPLATE_PATH:/etc/cadence-notification/config/docker.yaml

exec cadence-notification --root $CADENCE_NOTIFICATION_HOME --env docker start --services=$SERVICES

