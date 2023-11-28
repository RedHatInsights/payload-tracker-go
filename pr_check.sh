#!/bin/bash

# --------------------------------------------
# Options that must be configured by app owner
# --------------------------------------------
APP_NAME="payload-tracker"  # name of app-sre "application" folder this component lives in
COMPONENT_NAME="payload-tracker-go"  # name of app-sre "resourceTemplate" in deploy.yaml for this component
IMAGE="quay.io/cloudservices/payload-tracker-go"  

# ADD BACK IN WHEN PAYLOAD-TRACKER-GO HAS SMOKE TESTS
IQE_PLUGINS="payload-tracker"
IQE_MARKER_EXPRESSION="smoke"
IQE_FILTER_EXPRESSION=""
IQE_CJI_TIMEOUT="30m"
EXTRA_DEPLOY_ARGS="--single-replicas"

# Install bonfire repo/initialize
CICD_URL=https://raw.githubusercontent.com/RedHatInsights/bonfire/master/cicd
curl -s $CICD_URL/bootstrap.sh > .cicd_bootstrap.sh && source .cicd_bootstrap.sh

source $CICD_ROOT/build.sh
# source $APP_ROOT/unit_test.sh
source $CICD_ROOT/deploy_ephemeral_env.sh
oc rsh -n $NAMESPACE deployment/payload-tracker-api ./pt-seeder
COMPONENT_NAME=payload-tracker
source $CICD_ROOT/cji_smoke_test.sh
source $CICD_ROOT/post_test_results.sh
