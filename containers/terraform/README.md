

### Volumes
- terraform base (ro) or scenario terraform (ro)
- AWS credentials (ro)
- statefiles (rw)
- SSH key (w)

### Env vars
- TF_VAR_name="value"

```bash
docker run --rm -it --entrypoint sh -e TF_VAR_aws_profile=aaiken -e TF_VAR_aws_region=us-east-1 -e TF_PLUGIN_CACHE_DIR=/terraform-cache -w /mnt/base/aws -v $(pwd)/base/aws:/mnt/base/aws -v $HOME/.config/vmGoat/state:/mnt/state -v $HOME/.aws:/mnt/aws:ro hashicorp/terraform:latest
```

