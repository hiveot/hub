// This is a simple web element (no shadow dom) for displaying the device-type icon.
// This uses iconify-icon for the actual icon presentation.
// TBD: upload the map of device types to icon names?
//
// attributes:
//  deviceType: type of the icon
//

// FIXME: remove external dependency
import * as vocab from "../static/vocab.js"

const SensorIcon = "mdi:import"
const ServiceIcon = "mdi:cube-outline"
const ActuatorIcon = "mdi:export"
const ControllerIcon = "mdi:usb" //"molecule"
const ErrorIcon = "mdi:alert-circle"

// TODO: icons from config
const deviceTypeIcons = {
    [vocab.DeviceTypeBinarySwitch]: ActuatorIcon,
    [vocab.DeviceTypeBinding]: ServiceIcon,
    [vocab.DeviceTypeCamera]: SensorIcon,
    [vocab.DeviceTypeGateway]: ControllerIcon,
    [vocab.DeviceTypeService]: ServiceIcon,
    [vocab.DeviceTypeMultisensor]: SensorIcon,
    [vocab.DeviceTypeSensor]: SensorIcon,
    [vocab.DeviceTypeThermometer]: SensorIcon,
    // the following types should be changed in the binding to use the vocabulary
    "Binary Sensor": SensorIcon,
    "Binary Switch": ActuatorIcon,
    "Multilevel Sensor": SensorIcon,
    "Multilevel Switch": ActuatorIcon,
    "Static Controller": ControllerIcon,
    "error": ErrorIcon,
}

const template = `
    <iconify-icon icon></iconify-icon>
`

class HDeviceIcon extends HTMLElement {

    constructor() {
        super();
        this.innerHTML = template;
        this.iconEl = this.querySelector("[icon]")
    }

    static get observedAttributes() {
        return ["deviceType"];
    }

    attributeChangedCallback(name, oldValue, newValue) {
        // console.log("attributeChangedCallback: " + name + "=" + newValue);
    }

    connectedCallback() {
        let dt = this.getAttribute("deviceType")
        let iconName = deviceTypeIcons[dt]
        this.iconEl.setAttribute("icon", iconName)
    }
}

customElements.define('h-device-icon', HDeviceIcon)
