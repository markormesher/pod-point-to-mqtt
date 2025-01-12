[![CircleCI](https://img.shields.io/circleci/build/github/markormesher/pod-point-to-mqtt)](https://app.circleci.com/pipelines/github/markormesher/pod-point-to-mqtt)
[![Releases on GHCR](https://img.shields.io/badge/releases-ghcr.io-green)](https://ghcr.io/markormesher/pod-point-to-mqtt)

# Pod-Point to MQTT

This project uses the API that backs Pod-Point's mobile app to publish details about your home chargers, including their charging state, schedules, etc. Note that this is an **unofficial** API - Pod-Point could change it at any time, which would likely (temporarily) break this project.

## Configuration

Configuration is via environment variables:

- `MQTT_CONNECTION_STRING` - MQTT connection string, including protocol, host and port (default: `mqtt://0.0.0.0:1883`).
- `MQTT_TOPIC_PREFIX` - topix prefix (default: `pod_point`).
- `UPDATE_INTERVAL` - interval in seconds for updates; if this is <= 0 then the program will run once and exit (default: `0`).
- `POD_POINT_USERNAME` - your Pod-Point username (usually your email address).
- `POD_POINT_PASSWORD` - your Pod-Point password (see auth notes below).
- `DATA_DIR` - path to a secure directory where auth details can be persisted (see auth notes below).

## MQTT Topics

_For the full list, see [main.go](./cmd/main.go)._

- `${prefix}/_meta/last_seen` - RFC3339 timestamp of when the program last ran.
- `${prefix}/pods/${id}/state/...` - Pod details, for each Pod that you own.
  - `.../id` - numeric ID.
  - `.../pod_point_id` - text ID (e.g. PSL-123456).
  - `.../name` (may be blank).
  - `.../description` (may be blank).
  - `.../last_contact` - RFC3339 timestamp of when this Pod last checked in with Pod-Point servers.
  - `.../model/...` - **Model details**
    - `.../id`
    - `.../name`
    - `.../vendor`
    - `.../image_url`
  - `.../connectors/${id}/...` - **Connector details**, for each connector this Pod has.
    - `.../id` - numeric connector ID.
    - `.../door` - text door name.
    - `.../door_id` - numeric door ID.
    - `.../power` - max power, in kW.
    - `.../current` - max current, in A.
    - `.../voltage` - voltage, in V.
    - `.../charging_method` - method description (e.g. Single Phase AC).
    - `.../status` - connector status, see below for values.
    - `.../has_cable` - whether a cable is part of the unit (not whether a cable is _currently_ plugged in).
    - `.../socket/...` - **Socket details**
      - `.../type`
      - `.../description`
      - `.../ocpp_name` - [OCPP](https://openchargealliance.org/protocols/open-charge-point-protocol) name
      - `.../ocpp_code` - [OCPP](https://openchargealliance.org/protocols/open-charge-point-protocol) code
  - `.../charging/...` - **Charging details**
    - `.../mode` - overall charging mode, see below for values.
    - `.../allowed` - whether the Pod is allowing charging now.
    - `.../allowed_by_schedule` - whether the schedule permits charging now.
    - `.../schedules/${day}/...` - charging schedules, for days 1 to 7 (Monday to Sunday).
      - `.../start_time` - start time in HH:MM format.
      - `.../end_time` - end time in HH:MM format.
      - `.../active` - whether this status is currently active.
    - `.../override/...` - Override details.
      - `.../exists` - whether an override is currently in place.
      - `.../ends_at` - RFC3339 timestamp for the end of the override (blank if no override exists or the override is infinite).

## Explanation of Values

- The connector `status` values differ slightly between Pod models, which is why some of the statuses below overlap. The best thing to do is test out your own Pod and confirm which values it emits.
  - `available` or `idle` - the Pod is available for charging (although the schedule may stop it from doing so).
  - `charging` - the Pod is charging an EV.
  - `suspended-ev` - an EV is connected but has suspended the charge (because it is full, has its own schedule set, or is configured to stop at a given SoC).
  - `suspended-evse` - an EV is connected but the Pod has paused the charge (because of a schedule, the key lock, or balancing power with the house).
  - `pending` - an API command has been sent but the Pod hasn't updated yet (can take 5-10 minutes).
  - `waiting-for-schedule` - the Pod is waiting for a schedule to begin before it will start charging.
  - `connected-waiting-for-schedule` - an EV is connected, but the Pod is waiting for a schedule to begin before it will start charging.
  - `charge-override` - the "Charge Now" feature has been used to override a schedule.
  - `unavailable` - the Pod is unavailable.
  - `out-of-service` - the Pod is out of service.
- `.../charging/mode` summarises the current charging mode, with the following possible values:
  - `SCHEDULE` - the Pod is following the configured schedule (the mobile app generously calls this "smart" mode).
  - `OVERRIDE` - the Pod is in schedule mode, but the "Charge Now" feature has been used to override the schedule and force charging until a given time.
    - Details of the override are present in `.../charging/override/...` topics.
  - `MANUAL` - the Pod will charge whenever an EV is connected.
- `.../charging/allowed` summarises whether charging is allowed right now, taking into account the mode and schedules.
- `.../charging/allowed_by_schedule` reports whether the schedule would allow charging now, if the charger were in schedule mode.

## Auth Notes

- Providing your username and password as an auth mechanism sucks. As this is an unofficial API, there's not much we can do about that unfortunately.
- This project will not work if you have MFA enabled on your account ([#5](https://github.com/markormesher/pod-point-to-mqtt/issues/5)).
- If `DATA_DIR` is configured, this app will store a refresh token in a file there. This allows us to be good API citizens and refresh existing tokens, rather than generating new ones every time theapp is restarted.
  - Every _new_ token will trigger a "new login on your account" email; refreshed tokens will not.
  - Make sure this directory is secure - with an API or refresh token, someone can do damage to your Pod-Point account.
