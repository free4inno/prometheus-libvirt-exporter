# FROM alpine:3.17

# COPY ./.build/libvirt-exporter /bin/libvirt-exporter

# EXPOSE 9177

# RUN apk add --no-cache bash

# ENTRYPOINT ["/bin/libvirt-exporter"]

# Use a lightweight base image
FROM alpine:3.14

WORKDIR /root/

# Copy the pre-built binary from your host into the container
COPY .build/libvirt-exporter .

# Ensure the binary is executable
RUN chmod +x libvirt-exporter

# When mounting the volume, the socket at /var/run/libvirt/libvirt-sock on the host 
# will be available at the same path in the container. 
# Ensure your app has permissions to access the socket.

EXPOSE 9177

USER root

# Command to run the binary
ENTRYPOINT ["./libvirt-exporter"]

# docker buildx build --platform linux/amd64 --tag registry.cn-beijing.aliyuncs.com/hufu-dev/libvirt-exporter:dev-latest --push .