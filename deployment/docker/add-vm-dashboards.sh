#!/bin/sh
set -ex

mkdir -p /tmp/dashboards

cp /var/lib/grafana/dashboards/vm.json /tmp/dashboards/victoriametrics-vm-ds.json
cp /var/lib/grafana/dashboards/vmagent.json /tmp/dashboards/vmagent-vm-ds.json
cp /var/lib/grafana/dashboards/vmalert.json /tmp/dashboards/vmalert-vm-ds.json

for file in /tmp/dashboards/*.json; do sed -i 's/\"id\"\:\ \"prometheus\"/\"id\"\:\ \"victoriametrics-datasource\"/g' "$file"; done
for file in /tmp/dashboards/*.json; do sed -i  's/\"type\"\:\ \"prometheus\"/\"type\"\:\ \"victoriametrics-datasource\"/g' "$file"; done
for file in /tmp/dashboards/*.json; do sed -i  's/\"name\"\:\ \"Prometheus\"/\"name\"\:\ \"VictoriaMetrics\"/g' "$file"; done

for file in /tmp/dashboards/*.json; do sed -i  's/\"query\"\:\ \"prometheus\"/\"query\"\:\ \"victoriametrics-datasource\"/g' "$file"; done

for file in /tmp/dashboards/*.json; do sed -i  's/\"uid\"\:\ \"wNf0q_kZk\"/\"uid\"\:\ \"VMwNf0q_k\"/g' "$file"; done
for file in /tmp/dashboards/*.json; do sed -i  's/\"uid\"\:\ \"G7Z9GzMGz\"/\"uid\"\:\ \"VMZ9GzMGz\"/g' "$file"; done
for file in /tmp/dashboards/*.json; do sed -i  's/\"uid\"\:\ \"LzldHAVnz\"/\"uid\"\:\ \"VMldHAVnz\"/g' "$file"; done
for file in /tmp/dashboards/*.json; do sed -i  's/\"uid\"\:\ \"oS7Bi_0Wz\"/\"uid\"\:\ \"VM7Bi_0Wz\"/g' "$file"; done

for file in /tmp/dashboards/*.json; do sed -i  's/\"title\"\:\ \"VictoriaMetrics\"/\"title\"\:\ \"VictoriaMetrics - VM DS\"/g' "$file"; done
for file in /tmp/dashboards/*.json; do sed -i  's/\"title\"\:\ \"VictoriaMetrics - cluster\"/\"title\"\:\ \"VictoriaMetrics - cluster VM DS\"/g' "$file"; done
for file in /tmp/dashboards/*.json; do sed -i  's/\"title\"\:\ \"VictoriaMetrics\ -\ vmagent\"/\"title\"\:\ \"VictoriaMetrics\ -\ vmagent\ VM\ DS\"/g' "$file"; done
for file in /tmp/dashboards/*.json; do sed -i  's/\"title\"\:\ \"VictoriaMetrics\ -\ vmalert\"/\"title\"\:\ \"VictoriaMetrics\ -\ vmalert\ VM\ DS\"/g' "$file"; done

chown -R grafana:root /tmp/dashboards/

cp -r /tmp/dashboards/*.json /var/lib/grafana/dashboards