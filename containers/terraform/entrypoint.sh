#!/bin/sh

/bin/terraform init --reconfigure --upgrade

/bin/terraform apply --auto-approve
/bin/terraform output -json > /output.json
