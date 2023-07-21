### Snap integration

<https://snapcraft.io/>

snap link: <https://snapcraft.io/vmalert>

#### develop

Install snapcraft or docker

build snap package with command

 ```console
make build-snap
```

#### usage

installation and configuration:

```console
# install
snap install vmalert
# logs
snap logs vmalert
# restart
 snap restart vmalert
```

Configuration management:


```console
vi /var/snap/vmalert/current/etc/vmalert-alert-rules.yaml
```

after changes, you can trigger [config reread](https://docs.victoriametrics.com/vmalert.html#hot-config-reload) with `curl localhost:8880/-/reload`.


Configuration tuning is possible with editing extra_flags:

```console
echo 'FLAGS="-rule=/etc/vmalert-alert-rules.yaml -datasource.url=http://localhost:8428 -notifier.url=http://localhost:9093 -notifier.url=http://127.0.0.1:9093 -remoteWrite.url=http://localhost:8428 remoteRead.url=http://localhost:8428 -external.label=cluster=east-1 -external.label=replica=a' > /var/snap/vmalert/current/extra_flags
snap restart vmalert
```
See more at https://docs.victoriametrics.com/vmalert.html#quickstart
