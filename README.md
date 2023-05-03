# UD-CO2S-exporter
IODATA CO2 Sensor UD-CO2S prometheus exporter




## how to use

```bash

docker run -v /dev/ttyACM0:/dev/ttyACM0 -p 9233:9233 ghcr.io/bootjp/ud-co2s-exporter:latest

```

cli flags
```
usage: main [<flags>]


Flags:
  -h, --[no-]help              Show context-sensitive help (also try --help-long and --help-man).
      --device.name="/dev/ttyACM0"
                               Specify the UD-CO2S device path.(default /dev/ttyACM0)
      --exporter.addr=":9233"  Specifies the address on which the exporter listens (default :9233)

```