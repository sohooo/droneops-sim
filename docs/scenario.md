# Scenario DSL

The simulator can script high-level mission flows using a lightweight YAML-based domain specific language (DSL).
A scenario defines ordered **phases**, the **triggers** that move the mission between phases and the **enemy objectives** active in each phase.

## Example

```yaml
phases:
  - name: patrol
    enemy_objectives:
      - id: enemy-1
        action: patrol
    triggers:
      - event: time_elapsed
        value: 60
        next: intercept
  - name: intercept
    enemy_objectives:
      - id: enemy-1
        action: attack
        target: base
    triggers:
      - event: enemy_destroyed
        value: 1
        next: retreat
  - name: retreat
    enemy_objectives:
      - id: enemy-1
        action: retreat
```

* `event` describes what to wait for (`time_elapsed`, `enemy_destroyed`, etc.).
* `value` provides the threshold (seconds or count).
* `next` is the phase to transition to once the trigger condition is met.

## Loading

The scenario can be loaded at runtime using the `scenario` package:

```go
sc, err := scenario.Load("config/scenario.yaml")
```

Transitions are evaluated with `NextPhase`:

```go
next, ok := sc.NextPhase("patrol", scenario.Event{Type: "time_elapsed", Value: 60})
```

This DSL is intentionally simple and designed to be extended with additional trigger or objective types as the simulator evolves.
