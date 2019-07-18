# healthd

healthd is a service which receives monitoring reports from healthagents and persists those reports onto an HA etcd cluster.

Usage:
```
./healthd -etcd 192.168.1.1:2379               \
          -etcd 192.168.1.2:2379               \
          -etcd 192.168.1.3:2379               \
          -ca ./certs/ca.crt                   \
          -crt ./certs/server.example.org.crt  \
          -key ./certs/server.example.org.key. \
          -port 3443                           \
```

## License

Licensed under [MIT](./LICENSE).