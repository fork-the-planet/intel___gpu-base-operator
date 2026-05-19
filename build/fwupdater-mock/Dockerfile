# Copyright (c) 2026 Intel Corporation
# SPDX-License-Identifier: Apache-2.0
#
# Mock firmware updater image.
# Installs a fake 'xpu-smi' that validates arguments and simulates an update
# without touching any real hardware. Use this image as the 'updater' in a
# GPUFirmwareUpdate CR when you want to test the operator flow end-to-end on
# nodes that have no Intel GPUs or where real flashing is undesirable.
#
# Reference in CR:
#   spec:
#     updaterImage: intel/xpu-smi-mock:latest

FROM debian:stable-slim

COPY scripts/xpu-smi-mock.sh /usr/local/bin/xpu-smi
RUN chmod +x /usr/local/bin/xpu-smi
ENV FAKE_BDFS "0000:01:00.0 0000:02:00.0"

ENTRYPOINT ["/bin/sh"]
