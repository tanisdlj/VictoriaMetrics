### Snap integration

<https://snapcraft.io/>

snap link: <https://snapcraft.io/victorialogs>

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
snap install victorialogs
# logs
snap logs victorialogs
# restart
 snap restart victorialogs
```

Configuration tuning is possible with editing extra_flags:

```console
echo 'FLAGS="-storageDataPath=/var/lib/victorialogs -loggerFormat=json"' > /var/snap/victorialogs/current/extra_flags
snap restart victorialogs
```

Data folder located at `/var/snap/victorialogs/current/var/lib/victorialogs/`
