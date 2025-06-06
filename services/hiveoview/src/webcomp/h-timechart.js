/**
 * h-timechart.js is a simple time series chart component wrapper around chartjs.
 *
 * Intended to easily present sensor data as a line, area or bar graph without
 * having to wade through the expansive chartjs documentation.
 *
 * The default graph type is line, but setting a data series can overwrite this
 *
 * properties:
 * - chart-type: line, area, bar, scatter,
 * - chart-title: brand title text
 * - timestamp: end time of the history
 * - duration: time window to show in seconds
 *
 * Data element property overrides:
 * - unit: unit of measurement to include on the axis and legend
 * - key: unique key for this dataset. Used in 'addValue'.
 * - color: color override for this dataset
 * - chart-type: type of chart for this dataset
 */

// Note: chartjs requires import of a date library. date-fns or luxon will do
// import  '../static/chartjs-4.4.3.js'
// import  '../static/luxon-3.4.4.min.js'

export const PropChartType = "chart-type"
export const PropChartTitle = "chart-title"
export const PropTimestamp = "timestamp"
export const PropDuration = "duration"
export const PropStepped = "stepped"
export const PropNoLegend = "no-legend" // hide the legend

// Default colors for the datasets
export const DefaultColors =
    ["rgba(41,113,148,0.8)", "rgba(175,124,32,0.6)",
        "rgba(106,167,78,0.8)", "rgba(209,198,132,0.8)"]


// https://www.chartjs.org/docs/latest/configuration/responsive.html
const template = document.createElement('template')
template.innerHTML = `
    <div  style="position:relative; width:100%; height:100%; 
        display:flex; align-items:center; justify-content:center" 
    >
        <div id="_noDataText" style="font-style: italic; font-size: large; color: gray; font-weight: bold;">
            No data
        </div>
        <canvas id="_myChart" style="position:absolute; padding: 5px; margin:5px" ></canvas>
    </div>  
`

export class HTimechart extends HTMLElement {
    // value to highlight the y-axis row
    highlightYrow = 20;
    noLegend = false;

    // default chart configuration
    config ={
        type: 'line',
        // data is an array of objects with x, y fields: [ {x=1,y=3}, ...}
        data: {
            datasets: [{
                label: '',
                data: [],
                borderWidth: 1
            }]
        },
        options: {
            animation: {
                duration: 300
            },
            layout: {
              padding: {
                  right: 10
              }
              //   autopadding:false
            },
            // spanGaps: 1000*60*60*5,
            // IMPORTANT: in order to resize properly, set responsive to true
            // and disable maintainAspectRatio. Place the canvas in a div with
            // position:relative and set size on the div.
            // https://www.chartjs.org/docs/latest/configuration/responsive.html#configuration-options
            responsive: true,
            maintainAspectRatio: false,
            stacked:false,
            scales: {
                // Time axis
                x: {
                    // max: "2024-07-23T15:00:00.000-MST",
                    // max: "2024-07-23T15:00:00.000-07:00",
                    grid: {
                        // offset: true,
                        drawTicks: true,
                        // drawOnChartArea: true
                    },
                    border: {
                        display:true,
                        width:2,
                        dash: [2],
                        tickBorderDash: 10
                    },
                    // clip:false,
                    ticks: {
                        // maxTicksLimit: 24,
                        source: "auto",
                        stepSize:60,
                    },
                    type: 'time',
                    time: {
                        unit: 'minute',
                        tooltipFormat: 'yyyy-MM-dd HH:mm:ss',
                        displayFormats: {
                            second: 'HH:mm:ss',
                            minute: 'HH:mm',
                            hour: 'HH',
                        },
                        // round: "second"
                    }
                },
            },
            plugins: {
                datalabels: {

                },
                legend: {
                    display: false
                },
                title: {
                    display: true,
                    text: "title"
                },
                // https://www.chartjs.org/docs/latest/configuration/tooltip.html
                tooltip: {
                    intersect:false,
                    backgroundColor: "#3d415a",// "darkgray",
                    mode:'index',
                    padding: "10",
                    borderColor: "lightgreen",
                    borderWidth: "1"
                }
            }
        }
    }

    static get observedAttributes() {
        return [ PropChartType, PropChartTitle,
            PropTimestamp, PropDuration, PropNoLegend]
    }

    constructor() {
        super();

        const shadowRoot = this.attachShadow({mode: "open"});
        shadowRoot.append(template.content.cloneNode(true));
        let chartCanvas = shadowRoot.getElementById("_myChart")
        this.chartCanvas = chartCanvas.getContext("2d")

        this.chartType = 'line'
        this.chartTitle = undefined
        this.timestamp = luxon.DateTime.now().toISO()
        this.duration = -24*3600  // 1 day
        // this.innerHTML = template;
    }

    // addValue injects a new value without having to reload the whole chart.
    // if time is empty then use 'now'
    // key is the dataset key to add to. Default is the first one.
    addValue = (time, val, key) => {
        if (!time) {
            time = luxon.DateTime.now().toISO()
        }
// TODO: support for enums. For now just do binary true/false === 1/0
        if (val === "true") {
            val = 1
        } else if (val === "false") {
            val = 0
        }
//- end test

        let ds = this.config.data.datasets[0];
        // locate the dataset
        for (let i = 0; i < this.config.data.datasets.length; i++) {
            let hasKey = this.config.data.datasets[i].key === key;
            if (hasKey) {
                ds = this.config.data.datasets[i];
                break
            }
        }
        // insert the new value to the beginning of the data array
        ds.data.unshift({x:time,y:val})

        // update the end time to the new value and start time 24 hours before the end time
        let endTime = luxon.DateTime.fromISO(time)
        if (endTime > this.timestamp) {
            this.timestamp = endTime
        }
        let newStartTime = endTime.plus((this.duration-1800)*1000)
        let newEndTime = endTime.plus({minute:30})

        this.config.options.scales.x.min = newStartTime.toISO();
        this.config.options.scales.x.max = newEndTime.toISO();

        this.chart.update();
    }

    attributeChangedCallback(name, oldValue, newValue) {
        if (name === PropChartTitle) {
            this.chartTitle = newValue
        } else if (name === PropChartType) {
            this.chartType = newValue
        } else if (name === PropDuration) {
            this.duration = parseInt(newValue)
        } else if (name === PropNoLegend) {
            this.noLegend = newValue !== "true"
        } else if (name === PropTimestamp) {
            this.timestamp = newValue
        }
    }

    connectedCallback() {
        let readyEvent = new Event('on-timechart-ready', {
            bubbles:true,
            cancelable: false,
            composed: true
        })
        this.noDataEl = this.shadowRoot.getElementById("_noDataText")

        let dataElList = this.querySelectorAll('data')
        for (let i = 0; i < dataElList.length; i++) {
            let dataEl= dataElList[i]
            let dataPoints = JSON.parse(dataEl.innerText)
            let dataColor = dataEl.getAttribute('color')
            let dataKey = dataEl.getAttribute('key')
            let dataTitle = dataEl.getAttribute('title')
            let dataUnit = dataEl.getAttribute('unit')
            let steppedProp = dataEl.getAttribute(PropStepped)
            let stepped = (steppedProp?.toLowerCase()==="true")
            if (dataPoints) {
                this.setTimeSeries(i, dataKey, dataTitle,
                    dataPoints, dataUnit, stepped, dataColor)
            }
        }
        this.render();
        this.dispatchEvent(readyEvent);
    }

    disconnectedCallback() {
        if (this.chart) {
            this.chart.destroy();
            this.chart = null
        }
    }


    render = () => {
        let chartType = this.chartType
        // area charts are line charts with fill
        if (chartType === 'area') {
            chartType = 'line';
        }

        if (this.chart) {
            this.chart.destroy();
            this.chart = null;
        }
        // let chartCanvas = this.getElementById('_myChart').getContext("2d");
        // let chartCanvas = shadowRoot.querySelector("[canvas]").getContext("2d");
        this.config.type = chartType
        this.config.options.plugins.title = {
            display: (!!this.chartTitle),
            text:  this.chartTitle
        }
        this.chart = new Chart(this.chartCanvas, this.config);
    }

    // Insert or replace a chartjs dataset at the given index.
    //
    // Before render is called: changes will be used right away
    // After render was called: call this.chart.update() to apply the changes.
    //
    // @param nr is the index to insert it at
    // @param ds is the array of [{x:timestamp,y:value},...]
    setDataSet = (nr, ds) => {
        // NOTE1: horrible hack to show the last vertical grid line at the last timestamp
        // NOTE2: this expects the time range to be newest to oldest, the further in the older it gets
        // FIXME: the intended end time is needed in case of missing data
        this.config.data.datasets[nr] = ds;
        let endTime = luxon.DateTime.fromISO(this.timestamp)
        let startTime = endTime.plus(this.duration*1000)

        this.config.options.scales.x.min = startTime.toISO();
        this.config.options.scales.x.max = endTime.toISO();

        // this.config.options.scales.x.clip= false;

        // if there is no data then show the No-Data element
        if (ds.data.length > 0) {
            this.noDataEl.style.display = "none"
        } else {
            this.noDataEl.style.display = "flex"
        }
    }

    // Set a new time series to display.
    // This is a simple helper function for common use-case.
    //
    // call 'this.chart.update()' after render;
    //
    // @param nr is the series index, 0 for default, 1... for multiple series
    // @param label is the label of this series
    // @param timePoints is an array of: {x:timestamp, y:value} in reverse order (newest first)
    // @param stepped to show a stepped graph (boolean)
    // @param dataColor optional override of the default color
    setTimeSeries = (nr,key, label, timePoints, dataUnit, stepped, dataColor) => {

        // console.log("setTimeSeries; stepped=",stepped)
        // assign a color
        if (!dataColor) {
            if (nr < DefaultColors.length) {
                dataColor = DefaultColors[nr]
            } else {
                dataColor = DefaultColors[0]
            }
        }

        let yaxisID = dataUnit ? dataUnit : "default"
        // construct a replacement dataset
        // console.debug("setTimeSeries", timePoints.length, "items", "label=",label)
        let dataset = {
            key:key,
            // label: label,
            data: timePoints,
            borderWidth: 1,
            borderColor: dataColor,
            backgroundColor: dataColor,
            label: label + " " + dataUnit,

            // bar chart options
            barThickness:3,
            // barPercentage: 0.9,
            // grouped: false,

            // line chart options
            stepped: stepped?'after':false,

            // area chart options
            fill: (this.chartType === "area"),

            // stepped: 'after',
            // tension: stepped? 0 : 0.1,  // bezier curve tension
            yAxisID: yaxisID
        }

        // Setup the y-axis scale for this dataset
        // Scales are based on the data unit. Add a scale if it doesnt exist.
        let hasScale= this.config.options.scales[yaxisID]
        if (!hasScale) {
            // the first scale is at the left, additional scales are on the right
            let isFirstScale = (nr === 0);

            // default scale for numeric vertical axis
            let scale = {
                display: true,
                position: isFirstScale ? 'left' : 'right',
                grid: {
                    // only want the grid lines for one axis to show up
                    drawOnChartArea: isFirstScale,
                },
                // the ticks have the same color as the line
                ticks: {
                    // stepSize: 0.01,
                    precision: 2,   // todo: use precision from DataSchema, if available
                    color: dataColor,
                    callback: (val, index, ticks) => {
                        return val + " " + dataUnit
                    }
                }
            }
            // TODO: support for labels as y-axis values
            let axisIsLabel = false
            if (axisIsLabel) {
                scale.type = 'category'
                scale.labels = ['true', 'false']
            } else {
            }
            this.config.options.scales[yaxisID] = scale
        }

        // fixme: per dataset setting?
        // if (label) {
            this.config.options.plugins.legend.display = !this.noLegend
            this.config.options.plugins.legend.align = 'start' // start, center, end
        // } else {
        //     this.config.options.plugins.legend.display = false
        // }
        this.setDataSet(nr, dataset);
    }

}

// window.HChartJS = HTimechart
customElements.define('h-timechart', HTimechart)

