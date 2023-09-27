# Build the manager binary
FROM alpine:3.10

WORKDIR /etc/slurm
# Copy the Go Modules manifests
## see compiler-relative-cmd.sh
COPY slurm /etc/slurm/slurm
COPY munge /etc/slurm/munge
COPY configure.sh /etc/slurm/configure.sh

RUN chmod +x /etc/slurm/configure.sh