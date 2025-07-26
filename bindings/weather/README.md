# Weather Service Binding

## Objective

Provide a 'weather station' for current weather using one of the available providers.

## Status

This integration is in development and breaking changes should be expected.

## Summary

This integration requests the current weather for configured locations and publishes events on changes to environmental data. 

This periodically reads the current and forecasted weather for configured locations and weather provider. 

Supported providers:
- Open-Meteo
- Weather Underground [todo]
- Environment Canada [todo]

Current weather information includes:
- temperature
- relative humidity
- atmospheric pressure at sea level for the locations 
- wind direction
- wind speed
- alerts

- Forcast weather information includes:
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
