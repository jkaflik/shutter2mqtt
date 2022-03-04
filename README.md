# shutter2mqtt

The initial idea is to wrap stateless Somfy shutter engines with a little bit of logic. This tool bridges the `Shutter` to a MQTT broker.


## TODO list

- [x] `RelaysShutter` implementation (with few safety checks like `PairedRelay` or `RelayPool`)
- [x] organise packages, setup linter and CI
- [ ] load configuration (either by file or MQTT topic)
- [ ] unit tests
- [ ] container image and Helm chart
- [ ] automation module (layered logic, e.g. control by time or sun activity)