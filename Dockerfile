FROM gcr.io/distroless/static
ENTRYPOINT ["/shutter2mqtt"]
COPY shutter2mqtt /
