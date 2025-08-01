/**
 h-gauge is a wrapper around canvas-gauges.
 See: https://canvas-gauges.com/

 This:
 1. Fixes the declarative html canvas not showing
 2. Uses the hiveot style defaults including day/night mode.
 3. Resizes the canvas to the parent container

 Properties passed to the canvas-gauge instance:
 @param gauge-type: thermometer, barometer, compass, hygrometer
 @param override: override the canvas-gauge settings as json string
 @param linear: show as linear gauge instead of radial gauge
 @param low-range: what is considered low value (min-value to min-value + low-range)
 @param high-range: what is considered high value (max-value - high-range to max-value
 @param value: value to display
 @param max-value: override the gauge maximum value (required when unit differs from default)
 @param min-value: override the gauge minimum value (required when unit differs from default)
 */

export const PropGaugeType = "gauge-type"
export const PropOverride = "override"
export const PropValue = "value"
export const PropLinear = "linear"
export const PropUnits = "units"
export const PropMaxValue = "max-value"
export const PropMinValue = "min-value"

export const GaugeTypeBarometer = "barometer"
export const GaugeTypeCompass = "compass"
export const GaugeTypeHygrometer = "hygrometer"
export const GaugeTypeThermometer = "thermometer"

// preset configurations
const GaugePresetBarometer = {
    colorNeedle: 'rgb(103,74,46)',
    fontNumbers: "Courier New",
    // TOOD: auto calc highlights from min/max
    highlights: [
        {"from": 950, "to": 960, "color": "rgba(93,131,245,0.93)"},
        {"from": 1010, "to": 1040, "color": "rgba(241,195,89,0.8)"},
    ],
    highlightsWidth: 5,
    minValue: 950,   // standard at sea level is 1031 (depends on altitude)
    maxValue: 1040,  // highest ever recorded is 1084mb
    needleType: "arrow",
    title: "Barometer",
    units: "mbar",   // default unit when not overridden
}

const GaugePresetCompass = {
    colorNeedleShadowDown: "#222",
    colorNeedle: "rgba(240, 128, 128, 1)",
    colorNeedleEnd: "rgba(255, 160, 122, .9)",
    // colorPlate: 'rgba(170,168,167,0.38)',
    // colorPlateEnd: 'rgb(130,127,123)',


    highlights: [
        {"from": -1, "to": 1, "color": "rgba(245,204,93,0.93)"},
        {"from": 89, "to": 91, "color": "rgba(245,204,93,0.93)"},
        {"from": 179, "to": 181, "color": "rgba(245,204,93,0.93)"},
        {"from": 269, "to": 271, "color": "rgba(245,204,93,0.93)"},
    ],

    minValue: 0,
    maxValue: 360,
    majorTicks: ["N", "NE", "E", "SE", "S", "SW", "W", "NW", "N"],
    minorTicks: 22,
    needleCircleSize: 0.1, // avoid 0 as it rotates the box
    needleCircleOuter: false,
    needleType: "line",
    needleStart: 75,
    needleEnd: 99,
    needleWidth: 3,

    startAngle: 180,
    ticksAngle: 360,
    title: "Heading",
    units: "degrees"
    // valueBox: false,
}

const GaugePresetHygrometer = {
    borderMiddleWidth: 5,
    borderOuterWidth: 1,

    colorValueBoxBackground: 'transparent',
    colorValueText: 'rgba(95,98,104,0.72)',

    colorBorderMiddle: "#85816d",
    colorBorderMiddleEnd: "#464432",
    colorBorderOuter: "#262620",

    colorNeedle: 'rgb(46,60,103)',
    colorNeedleEnd: 'rgb(79,112,214)',

    fontValueStyle: "italic",
    highlights: [
        {"from": 0, "to": 30, "color": "rgba(201,191,171,0.93)"},
        {"from": 80, "to": 100, "color": "rgba(113,150,242,0.8)"},
    ],
    minValue: 0,
    maxValue: 100,

    startAngle: 45,
    ticksAngle: 270,

    title: "Hygrometer",
    unit: "%",
    valueBox: true,
    valueBoxStroke: 0,
    valueText: "relative humidity",
    valueTextShadow: false,
}
const GaugePresetThermometer = {
    barWidth: 3,  // % progress bar width
    barBeginCircle: 11,  // bulb size

    borderOuterWidth: 0,
    borderMiddleWidth: 0,
    borderInnerWidth: 0,

    colorBarProgress: "rgba(178,66,66,0.91)",   // bottom part of gauge bar
    colorNeedle: "red",
    colorNumbers: 'green',

    fontValueSize: 24,   // this is relative to the gauge size?
    // fontNumbersSize: 30,
    fontNumbersWeight: "bold",

    highlights: [
        {"from": -30, "to": -10, "color": "rgba(28,65,178,0.93)"},
        {"from": -10, "to": 0, "color": "rgba(89,133,241,0.8)"},
        // {"from": 0, "to": 15, "color": "rgba(239,207,151,0.5)"},
        // {"from": 15, "to": 25, "color": "rgba(231,134,134,0.5)"},
        {"from": 25, "to": 30, "color": "rgba(231,134,134,0.5)"},
        {"from": 30, "to": 40, "color": "rgba(236,69,69,0.5)"},
    ],
    minValue: -30,
    maxValue: 40,

    title: "Thermometer",
    units: "C",  // default when not set
    // value box
    valueDec: 1,  // decimal temperature
    valueInt: 1,

    //--- radial gauge specific options
    colorNeedleCircleInner: "rgba(61,111,234,0.8)",
    colorNeedleCircleInnerEnd: "rgba(61,111,234,0.8)",
    colorNeedleCircleOuter: "red",
    colorNeedleCircleOuterEnd: "red",
    needleCircleSize: 5,
}

const GaugePresets = {
    [GaugeTypeBarometer]: GaugePresetBarometer,
    [GaugeTypeCompass]: GaugePresetCompass,
    [GaugeTypeHygrometer]: GaugePresetHygrometer,
    [GaugeTypeThermometer]: GaugePresetThermometer,
}


// HGauge presents a radial or linear gauge on canvas
// The canvas is added as a child and the gauge is initialized in the connected callback.
class HGauge extends HTMLElement {
    // base gauge config
    baseConfig = {
        animation: true,
        animationRule: "linear",
        // animationRule: 'elastic',
        animationDuration: 1500,

        // barWidth: 5,  // % of progress bar width
        // colorBar: "grey",   // top part of the gauge bar
        // colorBarProgress: "rgb(202,63,63)",   // bottom part of gauge bar
        borders: true,
        borderOuterWidth: 0,
        borderMiddleWidth: 0,
        borderInnerWidth: 0,
        colorBorderShadow: "#989818",

        // colorBorderMiddle: "#85816d",
        // colorBorderMiddleEnd: "#464432",
        // colorBorderMiddle: "#696052",
        // colorBorderMiddleEnd: "#85816d",
        // colorBorderOuter: "#464432",
        // colorBorderOuterEnd: "#989818",

        colorMajorTicks: 'rgba(54,59,73,0.89)',
        colorMinorTicks: 'rgba(73,74,77,0.8)',

        colorPlate: 'rgb(170,168,167)',
        colorPlateEnd: 'rgb(130,127,123)',
        colorTitle: 'rgba(104,95,95,0.72)',
        colorUnits: 'rgba(104,95,95,0.72)',

        // colorNeedle: "red",
        // colorNumbers: 'var(--pico-color)',
        // colorNumbers: '#eee',
        // colorNumbers: 'green',

        // fontNumbersSize: 30,
        // fontNumbersWeight: "bold",
        // fontUnitsSize: 24,
        fontValueSize: 24,   // this is relative to the gauge size?
        //highlights: [],  // should be within min-max range
        highlightsWidth: 7,

        // majorTicksDec: 3,   doesn't work?
        // majorTicksInt: 3,   doesn't work?
        minorTicks: 10,
        // minValue: -30,
        // maxValue: 50,
        // majorTicks MUST match minValue and maxValue
        // majorTicks: [-30, -20, -10, 0, 10, 20, 30, 40, 50],
        needleWidth: 2,
        needleType: "arrow",
        stepSize: 10,   //for major ticks
        strokeTicks: true,
        type: "radial-gauge",

        // valueBox: false,
        // valueDec: 1,
        // valueInt: 1,
        // fontValue: // font family
        valueBoxStroke: 0,
        // fontValueStyle: "normal",

        //--- radial gauge specific options
        // colorNeedleCircleInner: "rgba(61,111,234,0.8)",
        // colorNeedleCircleInnerEnd: "rgba(61,111,234,0.8)",
        // colorNeedleCircleOuter: "red",
        // colorNeedleCircleOuterEnd: "red",
        // needleCircleSize: 5,
        // needleCircleInner: false,
        // needleCircleOuter: false,

        //--- linear gauge specific options
        // ticksWidth: 10, // length of major ticks (relative units)
        // barBeginCircle: 11,  // bulb size
        // numberSide: "right",
        // tickSide: "left",

    }


    static get observedAttributes() {
        // dynamically updatable
        return [PropGaugeType, PropLinear, PropMaxValue, PropMinValue, PropOverride, PropUnits, PropValue,]
    }

    constructor() {
        super()
        // this.innerHTML = template;

        this.canvasEl = document.createElement('canvas')
        this.canvasEl.id = this.id + "-canvas"
        this.appendChild(this.canvasEl)

        this.style.width = "100%"
        this.style.height = "100%"
        this.style.display = "block"
        this.gaugeType = GaugeTypeThermometer
        this.preset = GaugePresetThermometer
        this.override = ""
        this.isLinear = false // default to radial
        this.maxValue = 0  // min/max overrides when they differ
        this.minValue = 0
        this.unit = ""
        this.value = 0
    }

    // dynamic update of gauge values
    attributeChangedCallback(name, oldValue, newValue) {
        // console.log("attributeChanged, name=" + name + ", value=" + newValue)
        if (name === PropValue) {
            this.value = Number(newValue)
        } else if (name === PropMaxValue) {
            this.maxValue = Number(newValue)
        } else if (name === PropMinValue) {
            this.minValue = Number(newValue)
        } else if (name === PropGaugeType) {
            this.gaugeType = newValue
            this.preset = GaugePresets[this.gaugeType]
        } else if (name === PropOverride) {
            if (newValue) {
                try {
                    this.override = JSON.parse(newValue.toString())
                } catch (e) {
                    console.log("Preset override is not proper JSON: ", e.message)
                }
            }
        } else if (name === PropLinear) {
            this.isLinear = (newValue === "true")
        } else if (name === PropUnits) {
            this.unit = newValue
        }
        this.render()
    }

    // initialize and render the gauge
    connectedCallback() {
        // console.log("h-gauge connected, id=" + this.id + ", height=", this.offsetHeight)

        let config = Object.assign(this.baseConfig, this.preset, this.override)
        config.renderTo = this.canvasEl.id
        if (this.isLinear) {
            this.gauge = new LinearGauge(config)
        } else {
            this.gauge = new RadialGauge(config)
        }
        // overrides
        if (this.minValue !== this.maxValue) {
            config.minValue = this.minValue
            config.maxValue = this.minValue
        }

        // calc majorTicks based on minValue and maxValue
        let majorTicks = []
        // let range = config.maxValue - config.minValue
        for (let v = config.minValue; v <= config.maxValue; v += config.stepSize) {
            majorTicks.push(v)
        }
        this.gauge.update({majorTicks: majorTicks})
        this.render()

        // Adjust canvas size when widget resizes
        let debounceTimer = 0;
        this.sizeObserver = new ResizeObserver(()=>{
            clearTimeout(debounceTimer)
            debounceTimer = setTimeout(_=>{
                console.log("gauge resized. new width=",this.clientWidth, " height=",this.clientHeight)
                config={width:this.parentElement.clientWidth-5, height:this.parentElement.clientHeight-5}
                this.gauge.update(config)
            },100)
        });
        this.sizeObserver.observe(this.parentElement)
    }

    disconnectedCallback() {
        if (this.sizeObserver) {
            this.sizeObserver.disconnect()
            console.log("gauge disconnected")
            this.sizeObserver=undefined
        }
    }

    render() {
        if (!this.gauge) {
            return
        }
        // console.log("h-gauge render, id=" + this.id + ", height=", this.offsetHeight,
        //     ", parent height=", this.parentElement.offsetHeight)
        let config = Object.assign({}, this.baseConfig, this.preset, this.override)
        config.value = this.value
        if (this.unit) {
            config.units = this.unit
        }
        this.gauge.update(config)
    }

    // setValue(newValue) {
    //     console.log("h-gauge setValue=",newValue)
    //     this.value = Number(newValue)
    //     this.gauge.value = this.value
    //     this.gauge.update()
    // }
}

customElements.define('h-gauge', HGauge)

