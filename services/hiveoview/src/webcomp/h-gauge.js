/**
 * h-gauge is a wrapper around canvas-gauges.
 * See: https://canvas-gauges.com/
 *
 * This:
 *  1. Fixes the declarative html canvas not showing
 *  2. Uses the hiveot style defaults including day/night mode.
 *  3. Resizes the canvas to the parent container
 *
 * Properties passed to the canvas-gauge instance:
 * - type: "radial-gauge" (default) or "linear-gauge"
 * - value: value to display
 * - min-value: lower value
 * - max-value: upper value
 * - major-ticks: see canvas-gauges
 * - minor-ticks: see canvas-gauges
 * - unit: see canvas-gauges
 */

const PropMinValue = "min-value"
const PropMaxValue = "max-value"
const PropValue = "value"
const PropGaugeType = "type"
const PropUnit = "unit"


// HGauge presents a radial or linear gauge on canvas
// The canvas is added as a child and the gauge is initialized in the connected callback.
class HGauge extends HTMLElement {
    // default radial gauge config
    config = {
        animationRule: 'elastic',
        animationDuration: 500,

        barWidth: 5,
        // colorBar: "grey",   // top part of the gauge bar
        colorBarProgress: "rgb(202,63,63)",   // bottom part of gauge bar

        borders: false,

        colorNeedle: "red",
        // colorNumbers: 'var(--pico-color)',
        // colorNumbers: '#eee',
        colorNumbers: 'green',
        colorPlate: 'transparent',
        colorNeedleCircleOuter: "red",
        colorNeedleCircleOuterEnd: "red",
        colorNeedleCircleInner: "rgba(61,111,234,0.8)",
        colorNeedleCircleInnerEnd: "rgba(61,111,234,0.8)",
        colorMajorTicks: 'rgba(121,157,244,0.8)',
        colorMinorTicks: 'white',

        fontNumbersSize: 20,
        fontNumbersWeight: "bold",
        // fontUnitsSize: 24,

        // highlights doesn't display in the right spot!??
        highlights: [
            {"from": -20, "to": -10, "color": "rgba(66,66,231,0.5)"},
            {"from": -10, "to": 0, "color": "rgba(61,111,234,0.5)"},
            {"from": 0, "to": 15, "color": "rgba(239,207,151,0.5)"},
            {"from": 15, "to": 30, "color": "rgba(231,134,134,0.5)"},
            {"from": 30, "to": 40, "color": "rgba(236,69,69,0.5)"},
        ],
        minorTicks: 10,
        // minValue: -30,
        // maxValue: 50,
        // majorTicks MUST match minValue and maxValue
        // majorTicks: [-30, -20, -10, 0, 10, 20, 30, 40, 50],
        needleWidth: 2,
        needleCircleSize: 10,
        needleType: "arrow",
        strokeTicks: true,
        type: "radial-gauge",
        valueDec: 1,
        valueInt: 1,

        //--- radial gauge options

        //--- linear gauge options
        // ticksWidth: 10, // length of major ticks (relative units)
    }


    static get observedAttributes() {
        // dynamically updatable
        return [PropGaugeType, PropValue, PropMinValue, PropMaxValue, PropUnit]
    }

    constructor() {
        super()
        // this.innerHTML = template;

        this.canvasEl = document.createElement('canvas')
        this.canvasEl.id = this.id + "-canvas"
        this.appendChild(this.canvasEl)

        // this.canvasEl = this.getElementsByTagName("canvas")[0]
        // this.canvasEl.style.width = "100%"
        // this.canvasEl.style.height = "100%"
        this.minValue = 0
        this.maxValue = 100
        this.stepSize = 10
        this.value = 0
        this.unit = ""

        // const templateEl = document.createElement('template')
        // templateEl.innerHtml = template
        // const shadowRoot = this.attachShadow({mode: "open"});
        // shadowRoot.append(templateEl.content.cloneNode(true));
        this.gaugeType = "radial"
    }

    // dynamic update of gauge values
    attributeChangedCallback(name, oldValue, newValue) {
        console.log("attributeChanged, name=" + name + ", value=" + newValue)
        if (name === PropValue) {
            this.value = Number(newValue)
            newValue = this.value
        } else if (name === PropMinValue) {
            this.minValue = Number(newValue)
            newValue = this.minValue
        } else if (name === PropMaxValue) {
            this.maxValue = Number(newValue)
            newValue = this.maxValue
        } else if (name === PropGaugeType) {
            this.gaugeType = newValue
        } else if (name === PropUnit) {
            this.unit = newValue
        }
        this.render()
    }

    // initialize and render the gauge
    connectedCallback() {
        console.log("h-gauge connected, id=" + this.id + ", height=", this.offsetHeight)
        this.config["renderTo"] = this.canvasEl.id
        if (this.gaugeType === "linear-gauge") {
            this.gauge = new LinearGauge(this.config)
        } else {
            this.gauge = new RadialGauge(this.config)
        }
        // calc highlights from minValue and maxValue
        let highlights = [
            {"from": -30, "to": 0, "color": "rgba(0,0, 255, .3)"},
            {"from": 0, "to": 50, "color": "rgba(255, 0, 0, .3)"}
        ]
        // calc majorTicks based on minValue and maxValue
        let majorTicks = []
        let range = this.maxValue - this.minValue
        for (let v = this.minValue; v <= this.maxValue; v += this.stepSize) {
            majorTicks.push(v)
        }
        this.gauge.update({majorTicks: majorTicks})
        this.render()
    }

    render() {
        if (this.gauge) {
            this.gauge.update({
                "value": this.value,
                "minValue": this.minValue,
                "maxValue": this.maxValue,
                "units": this.unit,
                // "highlights":highlights,
            })
        }
    }

    // setValue(newValue) {
    //     console.log("h-gauge setValue=",newValue)
    //     this.value = Number(newValue)
    //     this.gauge.value = this.value
    //     this.gauge.update()
    // }
}

customElements.define('h-gauge', HGauge)

