# TOH IR

Universal IR remote for the Jolla Phone 2026: an IR transmitter and receiver on a TOH ("The Other Half") back cover, and the **IR Remote** app with remotes from a public database, universal remotes and learning from an original remote.

> [!WARNING]
> **Working prototype.** It runs on the [TOH DevKit](https://github.com/lorenz-graf/toh-devkit) breadboard cover, not on a cover of its own yet. Tested with one unit: it turned on a Grundig TV and a NexiGo Aurora Pro projector, and recording a button from a remote worked. Range isn't measured yet (expected to be short, see Hardware). Use at your own risk.

## What it does

- **Add remote:** pick your device from [Flipper-IRDB](https://github.com/UberGuidoZ/Flipper-IRDB) (about 1950 remotes, CC0) by category → brand → model, or search. Added remotes stay in your list
- **Buttons in their usual places:** power, D-pad, volume/channel, colour keys, media keys, number pad and air-conditioner keys go to fixed spots; other buttons go under "More buttons". Button names from the files are matched to spots by a list of common spellings (`phone/cmd/toh-ird/keys.go`)
- **Long press → Record new signal:** point the original remote at the back of the phone and press the button; the recording replaces that button. Recordings of known protocols are saved as codes, everything else as raw timings. **Blank remote:** record every button yourself
- **Universal remotes** (TV, projector, audio, air conditioner): Flipper Zero's universal files, downloaded when you add one (GPL-3.0, so not included here). Each tap sends the next code; **Worked** keeps the one your device reacted to. (Sending all codes at once switched a TV on and off repeatedly)
- **Protocols:** NEC, NECext, NEC42, Samsung32, RC5/RC5X, RC6 (mode 0), Sony SIRC (12/15/20 bit), Kaseikyo (Panasonic), RCA, Pioneer, and raw. Written from the protocols' public timing descriptions, using the Flipper `.ir` conventions for address and command so database files work unchanged. NEC, RC5, RC6, Samsung32 and SIRC were checked on hardware by receiving the board's own output

## Hardware

On the TOH DevKit's breadboard, with its RP2040-Zero (pin numbers as printed on the Zero):

| Module | Pin | Zero |
|---|---|---|
| IR receiver (38 kHz, 1838-type module) | OUT | 14 |
| | VCC | 28 |
| | GND | GND |
| IR transmitter (LED module) | DAT | 7 |
| | VCC | 3V3 |
| | GND | GND |

- **Never power the receiver from 5V.** These receiver modules pull their output up to their own supply, and the RP2040's pins only take 3.3 V. Since firmware 0.4 its VCC comes from pin 28 (3.3 V), which the firmware switches on only while learning, so the module's signal LED doesn't blink at every remote in the room
- The DevKit firmware uses pin 14 for an LED (and holds it low), so remove that LED; the IR firmware doesn't use it
- The transmitter module drives its LED straight from the pin (no transistor): only ~0.5–2 m. With the IR LED switched by an S8050 from 5 V (22 Ω, ~140 mA pulses; diagram `renders/ir-driver-wiring.png`) a TV reacted from about 10 m

## Firmware (`firmware/ir-fw/`)

Arduino sketch for the RP2040-Zero (core: Earle Philhower `rp2040:rp2040`, library: Adafruit NeoPixel). It contains the DevKit firmware (I²C target 0x42, same registers, so the DevKit app keeps working) plus:

- **IR registers** (map at the top of `ir-fw.ino`): send buffer (durations as 16-bit microseconds, mark first) + carrier frequency, learn command, buffer for the recorded frame, checksums for both directions
- **Fix for the DevKit's I²C lock-up:** a pending TX abort is cleared before answering a read. While the abort flag is set, the RP2040's TX FIFO discards writes, so the answer got lost and the chip held SCL low until a reset
- **USB serial console** (115200) for debugging: prints every received frame; `t` sends a test code, `s` repeats the last received frame, `l` lets the receiver listen while sending (to check what goes out), `r` keeps the receiver on to watch every frame

Build:

```
arduino-cli compile --fqbn rp2040:rp2040:waveshare_rp2040_zero --output-dir ../build/fw ir-fw
```

Flash: hold **BOOT** while plugging the Zero into the PC, copy `build/fw/ir-fw.ino.uf2` onto the RPI-RP2 drive. With the IR firmware running and the cover on the phone, `toh-i2c 0x42 w 0xf0 0xb0` (from the DevKit package) also puts the Zero into flash mode, if it's connected to the PC by USB.

## Phone package (`phone/`)

`./build-rpm.sh` builds `build/toh-ir-<version>-<release>.aarch64.rpm` (needs Go, Python 3 + Pillow, `rpmbuild`, and the GitHub CLI `gh` for the database index). It needs the `toh-devkit` package installed. Contents:

- **IR Remote** app (pure QML, `app/toh-ir.qml`)
- **toh-ird** (Go): session D-Bus service `io.github.lorenz_graf.TohIr`, started on demand, exits when idle. It encodes protocols, sends, learns, and keeps your remotes as Flipper `.ir` files in `~/.local/share/toh-ir/remotes/` (TOH IR's own data in `#@` comment lines). The method list is at the top of `cmd/toh-ird/main.go`. Also a command line for testing: `toh-ird probe`, `toh-ird code <protocol> <address> <command>` (hex), `toh-ird send <file.ir> <name>`, `toh-ird learn`
- **Database index:** the list of Flipper-IRDB files at a pinned commit; files are downloaded from that commit when you add a remote
- **symbiosis config** `devkit-ir.yaml`, installed next to the DevKit's `devkit.yaml` in `/usr/share/tohd-1/tohs/0000/0000/`. symbiosis merges the access lists of all YAML files in that folder, so this file only allows `toh-ird` on the bus

Install: tap the RPM in the file manager, then take the cover off and put it back on (symbiosis reads the cover's config on attach). Uninstall: `devel-su pkcon remove toh-ir`.

Tests (on the PC): `go test ./...` in `phone/` (protocol round trips). `IRDB_DIR=<folder with .ir files> go test -run Real -v ./internal/irproto` decodes raw recordings from real files.

## The phone's own IR driver

The Jolla Phone 2026's kernel has an IR transmitter driver (`mtk-pwm-ir-tx`, `/dev/lirc0`, send only), but every send fails with `ENOMEM`, and the specs list no IR LED. Most likely it's left over from MediaTek's reference design, so IR needs a cover.

## Licences

- This repo: BSD-3-Clause ([LICENSE](LICENSE))
- Remote database: [Flipper-IRDB](https://github.com/UberGuidoZ/Flipper-IRDB), CC0, downloaded per remote
- Universal remotes: [Flipper Zero firmware](https://github.com/flipperdevices/flipperzero-firmware) `infrared/assets/*.ir`, GPL-3.0, downloaded when you add one, not included

Made with the help of Claude.
