### Snap integration

<https://snapcraft.io/>

snap link: <https://snapcraft.io/vmagent>

#### develop

Install snapcraft or docker

build snap package with command

 ```console
make build-snap
```

It produces snap package with current git version - `victoriametrics_v1.46.0+git1.1bebd021a-dirty_all.snap`.
You can install it with command: `snap install victoriametrics_v1.46.0+git1.1bebd021a-dirty_all.snap --dangerous`

#### usage

installation and configuration:

```console
# install
snap install vmagent
# logs
snap logs vmagent
# restart
 snap restart vmagent
```

Configuration management:

 Prometheus scrape config can be edited with your favorite editor, its located at

```console
vi /var/snap/vmagent/current/etc/vmagent-scrape-config.yaml
```

after changes, you can trigger config reread with `curl localhost:8429/-/reload`.

Configuration tuning is possible with editing extra_flags:

```console
echo 'FLAGS="-remoteWrite.url=http://victoriametrics:8428/api/v1/write -remoteWrite.tmpDataPath=/var/lib/vmagent-remotewrite-data -promscrape.config=/etc/vmagent-scrape-config.yaml' > /var/snap/vmagent/current/extra_flags
snap restart vmagent
```

Data folder located at `/var/snap/vmagent/current/vmagent-remotewrite-data/`
