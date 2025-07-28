# Weather Service Binding

## Objective

Provide a 'weather station' for current weather using one of the available providers.

## Status

This integration is in development and breaking changes should be expected.

## Summary

This integration binding requests the current weather for configured locations and publishes events on changes to environmental data. 

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

Forcast weather information includes:
- 1 week forecast [todo]

## Configuration

Overrides for defaults are done in the weather.yaml configuration file. A default file is provided in the config folder.

Locations can be managed through the actions 'add/remove location' and by setting their properties for name, latitude and longitude. [work in progress] 

## Dependencies

This integration works with the [HiveOT Hub](https://github.com/hiveot/hub).
This binding needs an internet connection to access the [Open-Meteo API](https://open-meteo.com/en/docs) server.


## Usage

While this binding runs out of the box, some configuration is required:
- The config/weather.yaml file can be used to change defaults and add weather locations
- The binding actions can be used to add/remove locations 

On startup a TD (Thing Description) document is published in the HiveOT directory for the binding and each of the locations. Events notify of weather updates and binding actions can be used to manage the locations. 

Build and install with the hub plugins into the installation hiveot/plugins folder and start it using the launcher.


## Credits

A big thank-you to [Open-Meteo](https://open-meteo.com/) for providing an easy to use API that is free for non-commercial usage. For commercial usage see open-meteo pricing at https://open-meteo.com/en/pricing

