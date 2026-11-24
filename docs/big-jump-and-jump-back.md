# How big_jump and jump_back decide

## Scope

Holds for the handlers `big_jump_*` and `jump_back_*` in `pkg/handler`, which watch meter
totals: energy in kWh, water and gas volume in liters. Both assume a meter that only counts up
and reports its total rounded to a fixed step.

**Not this** for values that rise and fall by nature, such as power or temperature: neither
handler is registered for them, and a falling value is normal there.

All times are those at which the platform received a message (the Kafka message time), not
those at which the device measured. For devices whose messages regularly arrive late or
bundled, the big_jump test cannot be trusted, see [Known limits](#known-limits). Other handlers
were not checked against this document.

## Notation

Per device and service:

| Symbol | Meaning |
|---|---|
| `Z_i` | meter reading in message `i` |
| `t_i` | time the platform received message `i` |
| `d_i = Z_i − Z_{i−1}` | increment since the previous message |
| `Δt_i = t_i − t_{i−1}` | time since the previous message |
| `q` | resolution: the step the meter rounds to, for example 0.001 kWh |

A meter that rounds down reports `Z_i = q·⌊E(t_i)/q⌋` for its true total `E`. So `d_i` differs
from the true consumption between two messages by less than one step `q`.

## big_jump

### The test

After the learning phase, a message is reported as a big jump if

```
d_i > P_max · Δt_i + 2·q̂
```

- `q̂` estimates `q`: the smallest increment above zero seen so far.
- `P_max` is a high but still plausible counting speed of this meter, "power" for short, in the
  meter's unit per second: `P_max = 1.5 · Q99`, where `Q99` is the 99 % quantile of the window
  powers `R_k` of the last 7 days.

If the true power stays below `P_max` between two messages, then `d_i < P_max · Δt_i + q`. An
increment above this bound cannot come from consumption. The second step of the allowance is a
reserve for receive times that are slightly off.

### Window powers

`P_max` is estimated from windows, not from single messages, because rounding dominates the
power of a single message: one step of 1 Wh within 20 s looks like 180 W from a device drawing
40 W.

A window starts at a message (`t_a`, `Z_a`) and closes with the first message `j` for which
`Z_j − Z_a ≥ 10·q̂` or `t_j − t_a ≥ 30 min`. Its power is `R_k = (Z_j − Z_a) / (t_j − t_a)`, and
the next window starts at `j`. A window closing on ten steps is off by about one step, 10 %, at
most.

- Every window is learned, a jump included. The quantile keeps a rare jump from raising
  `P_max`, while a lasting change of the load is learned within a few windows.
- A falling reading (`d_i < 0`: a reset, or messages processed out of order) is neither tested
  nor learned. A new window starts there.
- Messages are tested once 100 windows of the last 7 days are known. At 40 W and a resolution of
  1 Wh, a window lasts about 15 minutes, so the learning phase takes about a day and at most
  50 hours.

### Constants

In `pkg/handler/big_jump.go`. They were chosen with the simulation described below, not
calibrated on real data.

| Constant | Value | Meaning |
|---|---|---|
| `bigJumpWindowSteps` | 10 | a window closes after this many steps, which bounds its rounding error to about 1/10 |
| `bigJumpMaxWindow` | 30 min | a window also closes after this time, for meters that count slowly |
| `bigJumpHorizon` | 7 days | older windows are forgotten |
| `bigJumpQuantile` | 0.99 | quantile of the window powers |
| `bigJumpPowerFactor` | 1.5 | `P_max` is this factor times the quantile |
| `bigJumpAllowanceSteps` | 2 | rounding allowance in steps |
| `bigJumpMinWindows` | 100 | learning phase, in windows within the horizon |
| `bigJumpMaxWindows` | 5000 | most windows stored per meter |
| `bigJumpTolerance` | 1e-9 | relative difference below which two readings count as equal, so float noise does not shrink `q̂` |

### State

Per device and service, the handler store keeps `handlerstore_big_jump_<device>_<service>_state`,
read and written for every message, and `…_windows`, read and written when a window closes. If
the state is lost, the meter goes through the learning phase again. The keys `…_mean`,
`…_stddev` and `…_num_datepoints` belong to the earlier test and are no longer read.

### Known limits

- **A load not seen in the last 7 days.** `Q99` only covers loads that occurred within the
  horizon. When a meter first draws far more than before, the first messages can be reported.
  In the simulation with loads of 200 and 2000 W, every false alarm fell into the first two days,
  when the high load occurred for the first time.
- **Short peaks.** At low load a window spans 15 minutes or more, so a short high load, such as
  a kettle on an otherwise quiet plug, can exceed `P_max`. Without further information, for
  example the reported momentary power, this cannot be told apart from a meter error.
- **Late or bundled delivery.** When messages are held back and arrive together, `Δt_i` is
  shorter than the time the meter counted. A receive time off by `V` seconds hides up to
  `P_max · V`, and at the moment of the message that cannot be told apart from a jump of that
  size. Without device timestamps the question is ill-posed. At low load this does not matter
  (40 W for 4 s is 0.04 Wh). At 2000 W, simulated bundles of up to 60 s every 30 minutes caused
  about 4 false alarms per day. Bundles of several minutes also shorten the windows, inflate
  `P_max` and make the test blind at high load.
- **Shorter memory under the storage bound.** A meter that closes a window more often than
  every 2 minutes (7 days / 5000) remembers less than 7 days, for example about 2.3 days at
  2000 W with a message every 40 s, each of which closes a window.

### The simulation

Four load profiles, each with 5 runs of 14 days: the load switches between a base and a high
level every 0.5 to 4 hours, the counter is rounded down to 1 Wh, messages come every 20 to 60 s
while the load fluctuates and otherwise only every TelePeriod (Tasmota's periodic report). Each
run carries 5 injected jumps of +0.05 kWh and 5 of +0.5 kWh. The simulation is not part of this
repository.

| Profile | False alarms per day | +0.05 kWh found | +0.5 kWh found |
|---|---|---|---|
| 40/95 W, TelePeriod 300 s | 0 | 25 of 25 | 25 of 25 |
| 40/95 W, TelePeriod 3600 s | 0 | 24 of 25 | 25 of 25 |
| 200/2000 W, TelePeriod 300 s | 0.14 | 23 of 25 | 25 of 25 |
| 5/2000 W, TelePeriod 300 s | 0 | 23 of 25 | 25 of 25 |

The earlier test raised 2.2 to 13.7 false alarms per day on the same profiles. The
implementation was checked with 20 further simulated weeks per profile: no false alarm at 40 W,
95 W and 40/95 W; 39 in 140 days at 200/2000 W, the first of each week within its first two
days.

### Rejected alternatives

| Alternative | Why not |
|---|---|
| The earlier test: `d_i > mean + 5·std` of all increments ≠ 0 | It ignores `Δt_i`, so a message that came late after a quiet period exceeded the threshold: 2.2 to 13.7 false alarms per day. Negative increments entered the statistics, and after a reset the test stayed blind |
| The earlier test plus a minimum jump of 10 steps | Fixes only the reporting pattern it is tuned to: no false alarms with TelePeriod 300 s, but 2.2 per day with 3600 s and 13.7 at 200/2000 W |
| Rate since the last change of the reading, with mean + 5·std | Rounding noise of `q/Δt` dominates: 0.1 to 2.6 false alarms per day, +0.05 kWh found 3 to 19 of 25 |
| Not learning windows that contain an alarm | A lasting change of the load is never learned and stays in alarm: up to 317 false alarms per day |
| `P_max = mean + 5·std` of the window powers, over all time or the last 7 days | A real jump inflates `std` and blinds the test: +0.05 kWh found 9 to 23 of 25, against 23 to 25 with the quantile. Forgetting after 7 days limits how long a jump weighs, not how much |
| Window powers capped at the threshold before learning | 4.3 false alarms per day at 200/2000 W |
| `Q95` instead of `Q99`, or factor 2 instead of 1.5 | `Q95`: 0.61 false alarms per day at 200/2000 W. Factor 2: +0.05 kWh found only 15 to 16 of 25 at high load |
| Windows lasting at least 5 minutes, to store fewer | A new high load is learned more slowly: 0.83 instead of 0.14 false alarms per day at 200/2000 W |
| Allowance from the delivery delay measured with device timestamps | This service gets no device timestamp in general, and a delay measured on one device type does not carry over to others |
| Testing a bundled message against the message before the bundle | Halves the false alarms from bundled delivery only, 4.1 to 2.3 per day at 2000 W: the first punctual message after a late one is compressed too and cannot be recognized as part of a bundle |
| Allowance learned per meter, as a quantile of `d_i − P_max · Δt_i` | Reduces those false alarms to 0.8 per day, but finds +0.05 kWh only 16 instead of 23 of 25 and starts testing later |

## jump_back

A falling reading is reported only once the next reading confirms it. For the last three
readings `a, b, c`, oldest first, the handler reports if `b < a` and `c < a`.

- A reading processed out of order, such as 7.446, 7.445, 7.447, is followed by one at or above
  the level before the drop: not reported.
- After a reset, such as 33.498, 0, 0.001, the meter stays below that level: reported, one
  message later than with a comparison of two readings.

Rejected: sorting the readings by receive time. Two readings that were processed in swapped
order had arrived 44 ms apart, and where the order flips is not known, so it is unclear whether
the receive time would even show the right order.

## After a rollout

big_jump learns from scratch: for about one to two days per meter it reports nothing.
