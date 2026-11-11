# Bambu Lab LAN

RankOS uses one LAN MQTT client for every current Bambu family. The client does not branch on serial prefix. Discovery labels a printer only from the model text in its SSDP announcement. A serial alone stays a generic "Bambu Printer".

## Connection (all families)

| | |
|---|---|
| Transport | MQTT over TLS, port 8883 |
| Username | `bblp` |
| Password | The printer's access code |
| Report topic | `device/<serial>/report` |
| Request topic | `device/<serial>/request` |
| Protocol | MQTT 3.1.1, clean session |
| Client id | Random, 23 bytes or fewer, new for each session |
| TLS | The printer certificate is self-signed and its common name is the serial, not the host. The client skips verification on the LAN connection. |
| Full snapshot | `{"pushing":{"sequence_id":"0","command":"pushall"}}` |
| After a silent subscribe | `{"pushing":{"sequence_id":"0","command":"start"}}` |

`pushall` is the payload used by Home Assistant's Bambu integration for every model that project supports. Extra fields are not sent.

Reports are often partial. RankOS merges only the JSON keys that are present, so a later delta does not clear temperatures or the file name from an earlier snapshot. Numeric fields sent as strings are accepted. That string form was described in public protocol notes for A1-class reports; the parser accepts it for every family rather than special-casing one model.

`BAMBU_MQTT_DEBUG=1` also subscribes to `device/+/report` and logs the topic plus the first 300 bytes of each message.

## What to turn on at the printer

Bambu's Developer Mode instructions (H2 series, P2S, X series, P series, and A series) are:

1. Enable **LAN Only**.
2. Enable **Developer Mode**, and accept the notice.

Bambu's page says this opens the MQTT channel, the live view, and FTP, and the printer cannot stay bound to Bambu Cloud.

An older Bambu note on third-party authorization says status pushes keep working and that authorization blocks control (start a print, motion, temperatures, fans, AMS, camera) unless Developer Mode is on. If RankOS connects and then logs that no report arrived, enable both settings anyway. That is the configuration Bambu documents for third-party MQTT, and a wrong serial produces the same silence.

The access code is on the printer's network screen. It is not the cloud password.

## Families

| Family | Models | What public sources support |
|---|---|---|
| P1 | P1P, P1S | Same client as the others. Home Assistant and public protocol notes treat them as delta reporters: ask for `pushall`, then merge changes. |
| P2 | P2S | Bambu's Developer Mode page groups P2S with the H2 series (LAN Only, then Developer Mode). Home Assistant lists P2S. No public note of a different topic or `pushall` body. |
| A1 | A1, A1 mini | Public protocol notes from an A1 mini capture: full snapshot when `print.msg` is 0, deltas when it is 1, both with `command` `push_status`. Fan speeds in that capture were JSON strings. Two read-only clients were connected at once; that capture did not show the broker enforcing a single client or a unique client id. |
| A2 | A2 | No separate packet capture or model page was found. RankOS still uses the shared client. Do not assume a different topic. |
| H2 | H2D, H2S, H2C | Developer Mode steps are the H2/P2S section of Bambu's wiki. Home Assistant treats H2D, H2C, and H2D Pro as dual-nozzle machines, and H2D/H2S cameras as RTSP on the LAN live-view port rather than the MQTT status channel. RankOS reads `nozzle_temper` and `nozzle_target_temper` and does not model a second nozzle. AMS trays in the report are merged when the `ams` object is present. |
| H3 | H3 | Not described in the Developer Mode wiki sections or the Home Assistant model list that were checked. The shared client is used. Whether H3 needs any setting beyond LAN Only and Developer Mode was not verified. |
| X1 | X1, X1C, X1E | Included. Home Assistant and Bambu's authorization note cover X1. Reports are merged the same way, so a fuller X1 snapshot and a later partial update both apply. |

Concurrent-client behavior was only described for the A1-class capture above. It was not verified for P2S, H2S, A2, or H3. RankOS still uses a unique client id so a second session is not forced to share one.

Control commands (pause, resume, cancel, temperatures, fans, AMS, starting a print) follow the same MQTT session. On current firmware those commands are what Developer Mode is documented to allow. Status fields RankOS shows come from the report stream, not from a per-model table.
