FROM hashicorp/terraform:latest

RUN mkdir /terraform-cache

COPY --chmod=0500 /entrypoint.sh /entrypoint.sh

ENTRYPOINT [ "/entrypoint.sh" ]
