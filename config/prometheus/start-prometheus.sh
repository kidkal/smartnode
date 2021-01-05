#!/bin/sh
exec /bin/prometheus --storage.tsdb.path=/prometheus \
                     --web.console.libraries=/usr/share/prometheus/console_libraries \
                     --web.console.templates=/usr/share/prometheus/consoles \
                     --config.file=/setup/prometheus.yml
