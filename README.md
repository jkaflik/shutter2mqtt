# shutter2mqtt

The initial idea is to wrap stateless Somfy shutter engines with a little of logic. This tool bridges the `Shutter` to a MQTT broker.


## TODO list

- [ ] move this list to Github's issues
- [x] `RelaysShutter` implementation (with few safety checks like `PairedRelay` or `RelayPool`)
- [x] organise packages, setup linter and CI
- [x] load config from config.yaml or ENV vars
  - [ ] mute configuration by MQTT topic
- [ ] unit tests
  - [ ] `internal/mqtt/`
  - [ ] `internal/shutter/driver/relay/wired.go`
  - [ ] `internal/shutter/driver/relay/shutter.go`
- [ ] container image
- [ ] Helm chart
- [ ] automation module (layered logic, e.g. control by time or sun activity)
