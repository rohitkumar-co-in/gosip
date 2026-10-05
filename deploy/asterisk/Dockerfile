FROM ubuntu:24.04
ENV DEBIAN_FRONTEND=noninteractive PYTHONUNBUFFERED=1
RUN apt-get update && apt-get install -y --no-install-recommends asterisk python3 ca-certificates \
    && rm -rf /var/lib/apt/lists/* /etc/asterisk/* \
    && mkdir -p /etc/asterisk /var/run/asterisk /var/lib/gosip-pbx /usr/lib/asterisk \
    && ln -s "$(find /usr/lib -type d -path '*/asterisk/modules' -print -quit)" /usr/lib/asterisk/modules
COPY deploy/asterisk/bridge.py /opt/gosip/bridge.py
EXPOSE 5061/tcp 10000-10100/udp
ENTRYPOINT ["python3", "/opt/gosip/bridge.py"]
