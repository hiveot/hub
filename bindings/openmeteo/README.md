# Open-Meteo Integration

## Objective

Provide a 'weather station' for current weather using the Open-Meteo provider.

## Status

This integration is in development and breaking changes should be expected.

## Summary

The open-meteo integration requests the current weather for configured locations and publishes events on changes to environmental data. 

This supports tracking the following parameters for select locations:
- temperature
- relative humidity
- atmospheric pressure at sea level for the locations 
- wind direction
- wind speed
- alerts
- short term forecast

Configurations for:
- locations
- polling interval

## Dependencies

This integration works with the [HiveOT Hub](https://github.com/hiveot/hub).


## Usage

While this integration runs out of the box, some configuration is required:
- This integration requires one or more locations whose weather information to capture. 
- A 'Thing Definition' is generated for each valid location. The TD includes properties to enable/disable supported parameters.

Build and install with the hub plugins into the installation hiveot/plugins folder and start it using the launcher.

Out of the box it will use the meteo api endpoint. An API-key is not needed for non-commercial usage.

For commercial usage see open-meteo pricing at https://open-meteo.com/en/pricing

## Credits

A big thank-you to [Open-Meteo](https://open-meteo.com/) for providing an easy to use API that is free for non-commercial usage.
