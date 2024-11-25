# Cron Matcher Caddy Plugin

The **Cron Matcher** plugin for [Caddy](https://caddyserver.com) allows you to match HTTP requests based on cron expressions. It enables you to define time windows during which certain routes or handlers should be active. This plugin is useful for scheduling maintenance windows, rate limiting during peak hours, or any time-based request handling logic. By leveraging cron expressions, you can precisely control when specific request matchers are active, integrating flexible scheduling into your Caddy configurations.

[![Go Report Card](https://goreportcard.com/badge/github.com/steffenbusch/caddy-cron-matcher)](https://goreportcard.com/report/github.com/steffenbusch/caddy-cron-matcher)

## Features

This plugin introduces a new matcher that can be used within Caddy configurations:

- **`cron` matcher**: Matches requests based on time windows defined by cron expressions.

### Key Capabilities

- **Time-Based Request Matching**: Activate or deactivate request handlers based on precise time schedules.
- **Flexible Scheduling**: Use cron expressions to define complex schedules, including specific times, ranges, and intervals.
- **Dynamic Control**: Enable features like maintenance modes, time-limited promotions, or access restrictions during specific hours.
- **Multiple Schedule Support**: Combine multiple `cron` expressions to create OR-based matching.

## Building

To build Caddy with this module, use [xcaddy](https://github.com/caddyserver/xcaddy):

```bash
$ xcaddy build --with github.com/steffenbusch/caddy-cron-matcher
```

## Caddyfile Configuration

The cron matcher requires two cron expressions:

- **`enable_at`**: Specifies when the matcher should start matching requests.
- **`disable_at`**: Specifies when the matcher should stop matching requests.

Both expressions must be valid cron expressions and are required. Multiple cron matchers within a named matcher will be OR'ed together.

**Note on Time Matching Behavior**:
The `CronMatcher` considers `now` as inclusive when it aligns exactly with the `enable_at` time. This means that if the current time precisely matches the `enable_at` expression, it is treated as valid and within the active matching window. The `disable_at` time marks the end of the range and is not included in matches.

### Syntax

```caddyfile
@timedRequest cron <enable_at> <disable_at>
```

Multiple entries can be defined like this:

```caddyfile
@timedRequest {
    cron <enable_at> <disable_at>
    cron <enable_at_2> <disable_at_2>
}
```

### Examples

#### Example 1: Active Between 10:00 and 11:00 Daily

```caddyfile
@timedRequest cron "0 10 * * *" "0 11 * * *"

handle @timedRequest {
    respond "This route is active between 10:00 and 11:00."
}

handle {
    respond "This route is active outside of 10:00 to 11:00."
}
```

#### Example 2: Maintenance Mode on Sundays

```caddyfile
@maintenanceMode cron "0 0 * * SUN" "0 0 * * MON"

handle @maintenanceMode {
    respond "Maintenance mode is active."
}

handle {
    reverse_proxy localhost:8080
}
```

In this example, the maintenance mode is activated every Sunday at 00:00 and deactivated on Monday at 00:00.

#### Example 3: Multiple OR'ed Cron Schedules for Complex Maintenance

```caddyfile
@maintenance {
    cron "45 22 * * 1-5" "15 23 * * 1-5"
    cron "15 23 * * 0,6" "45 23 * * 0,6"
}

handle @maintenance {
    respond "It's in Maintenance"
}

handle {
    respond "No Maintenance"
}
```

In this example, the route matches any of the defined time windows. Maintenance is active from 22:45 to 23:15 on weekdays and from 23:15 to 23:45 on weekends.

#### Example 4: Special Promotion During Business Hours

```caddyfile
@businessHours cron "0 9 * * 1-5" "0 17 * * 1-5"

handle @businessHours {
    respond "Welcome! Enjoy our special business hours promotion."
}

handle {
    reverse_proxy localhost:8080
}
```

In this example, a promotional message is displayed during business hours from 09:00 to 17:00, Monday through Friday. Outside of these hours, requests are proxied to the backend application. This setup is ideal for displaying special offers or notices during operating hours.

### Error Handling

- **Both `enable_at` and `disable_at` are required**: If either is missing, Caddy will fail to start, and an error will be logged.
- **Invalid Cron Expressions**: If an invalid cron expression is provided, Caddy will fail to start, and an error indicating the invalid format will be logged.

### Notes

- **Time Zone Considerations**: Cron expressions are evaluated based on the server's local time zone. Ensure that your server's time zone is configured correctly to match your intended schedule.
- **Cron Expression Format**: The cron expressions follow the standard format:

```text
* * * * *
│ │ │ │ │
│ │ │ │ └─ Day of the week (0-6 or SUN-SAT)
│ │ │ └─── Month (1-12 or JAN-DEC)
│ │ └───── Day of the month (1-31)
│ └─────── Hour (0-23)
└───────── Minute (0-59)
```

#### Advanced Cron Expressions

While the commonly used 5-segment cron format (`<minute> <hour> <day> <month> <weekday>`) is supported, the underlying library ([Gronx](https://github.com/adhocore/gronx)) provides extended functionality with a 6-segment style that includes `<year>`. This format allows for more precise scheduling:

- A complete cron expression consists of 7 segments: `<second> <minute> <hour> <day> <month> <weekday> <year>`.
- For a 6-segment expression, if the 6th segment matches `<year>` (e.g., at least 4 digits), it is interpreted as `<minute> <hour> <day> <month> <weekday> <year>`. A default value of `0` is used for `<second>`.

Additionally, Gronx supports advanced modifiers for the `<day>` and `<weekday>` segments:

- **`L` (Last)**:
  - In the `<day>` segment, `L` represents the last day of the month (e.g., `L` could mean February 29th in a leap year).
  - In the `<weekday>` segment, `L` refers to the last occurrence of a specific weekday in a month (e.g., `2L` means the last Tuesday of the month).
- **`W` (Weekday)**:
  - For `<day>`, `W` specifies the closest weekday (Monday to Friday) to a given day (e.g., `10W` means the closest weekday to the 10th of the month).
- **`#` (Nth weekday)**:
  - In the `<weekday>` segment, `#` denotes the nth occurrence of a specific day in a month (e.g., `1#2` means the second Monday of the month).

For a complete overview of these features, refer to the [Gronx README](https://github.com/adhocore/gronx#readme).

## License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.

## Acknowledgements

- [Caddy](https://caddyserver.com) for providing a powerful and extensible web server.
- [Gronx](https://github.com/adhocore/gronx) for cron expression parsing and time calculations, used under the MIT License.
