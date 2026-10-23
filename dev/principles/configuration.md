# Configuration Is Documentation

> **Declare it. Document it. Expose it.**

Configuration is operational knowledge about a system. It is not an implementation detail,
a convenience mechanism, or a substitute for source code constants. It is an explicit
contract between designers, developers, operators, and runtime environments. Good
configuration makes behavior visible, understandable, controllable, and explainable.

## Principle

**Behavior that operations can influence belongs in configuration, not in code.**

- Every parameter that influences runtime behavior is explicitly configurable: thresholds,
  timeouts, retry policies, buffer sizes, resource limits, feature switches, security
  settings, retention periods, prediction horizons, expiration intervals.
- No hidden defaults. A parameter that shapes runtime behavior is supplied explicitly.
- Missing required configuration is detected early and reported clearly.
- Every parameter documents its purpose, expected type, allowed values, units, and
  operational impact, so a file explains both what a parameter does and why it exists.
- Applications define what can be configured. Operations define which values are used.
- Operationally significant constants are never hidden inside application logic.
- A running system exposes which configuration was supplied and which is currently active.
- Configuration is discoverable by humans and machines, because system intent stays
  explicit.

## Development Rule

**No hidden value may change system behavior.**

If a value influences runtime behavior, move it into configuration.

If a value lives in configuration, document what it does and why it exists.

If a required value is absent, fail at startup and name the missing parameter.

If behavior changed, configuration must be one of the places where the cause can be found.

## Guiding Rule

Nobody should have to read source code to understand operational behavior.

Every operationally significant parameter is:

    Explicitly configurable
    Explicitly documented
    Explicitly visible
    Explicitly owned

Configuration exposes operational assumptions rather than hiding them.
