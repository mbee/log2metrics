# log2metrics

- send logs over UDP
- receive logs over UDP and analyze them to expose prometheus-compatible metrics

Example:

```sh
➜ go run cmd/analyze/analyze.go -config ./templates/dpkg.yaml -syslogport 8888 # to listen to logs and expose prometheus' metrics
➜ go run cmd/sendlog/sendlog.go -log /var/log/dpkg.log.1 -syslogurl udp://localhost:8888 # to send logs
```

in another terminal:

```
➜ curl -s localhost:3054/metrics | head
# HELP dpkg_dpkg01_total The total number of lines matching regex ^(?P<date>....-..-..) (?P<time>..:..:..) status .+$
# TYPE dpkg_dpkg01_total counter
dpkg_dpkg01_total 1429
# HELP dpkg_dpkg02_total The total number of lines matching regex ^(?P<date>....-..-..) (?P<time>..:..:..) upgrade .+$
# TYPE dpkg_dpkg02_total counter
dpkg_dpkg02_total 104
# HELP dpkg_dpkg03_total The total number of lines matching regex ^(?P<date>....-..-..) (?P<time>..:..:..) startup .+$
# TYPE dpkg_dpkg03_total counter
dpkg_dpkg03_total 62
# HELP dpkg_dpkg04_total The total number of lines matching regex ^(?P<date>....-..-..) (?P<time>..:..:..) (?P<others>\\w+) .+$
```
