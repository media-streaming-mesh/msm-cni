FROM alpine:latest

RUN apk --no-cache add ca-certificates

LABEL description="MSM CNI plugin installer."

COPY msm-cni /opt/cni/bin/
COPY msm-iptables /opt/cni/bin/
COPY installer /usr/local/bin/

ENV PATH=$PATH:/opt/cni/bin
WORKDIR /opt/cni/bin
CMD ["/usr/local/bin/install"]
