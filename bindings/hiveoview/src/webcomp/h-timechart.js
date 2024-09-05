/**
 * h-timechart.js is a simple time series chart component wrapper around chartjs.
 *
 * Intended to easily present sensor data as a line, area or bar graph without
 * having to wade through the expansive chartjs documentation.
 *
 * The default graph type is line, but setting a data series can overwrite this
 *
 * properties:
 *  chart-type: line
 *  chart-title: brand title text
 *  timestamp: end time of the history
 *  duration: time window to show in seconds
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
                x: {
                    max: "2024-07-23T15:00:00.000-07:00",
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
                    ticks: {
                        // maxTicksLimit: 24,
                        source: "auto"
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
                // y: {
                //     grid: {
                //         // test to highlight the row of a certain value
                //         color: (ctx)=> {
                //             return (ctx.tick.value === this.highlightYrow)?"green":'rgba(0,0,0,0.1)'
                //         }
                //     },
                //     ticks: {
                //         maxTicksLimit: 30,
                //         source: "auto"
                //     }
                // },
                // y1: {
                //     // type: 'linear',
                //     display:false,
                //     position: 'right',
                //     ticks: {
                //         maxTicksLimit: 30,
                //         source: "auto"
                //     },
                //     grid: {
                //         drawOnChartArea: false, // only want the grid lines for one axis to show up
                //     },
                // }
            },
            plugins: {
                legend: {
                    display: false
                },
                title: {
                    display: true,
                    text: "title"
                },
                tooltip: {
                    intersect:false,
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
            let dataKey = dataEl.getAttribute('key')
            let dataTitle = dataEl.getAttribute('title')
            let dataUnit = dataEl.getAttribute('unit')
            let steppedProp = dataEl.getAttribute(PropStepped)
            let stepped = (steppedProp?.toLowerCase()==="true")
            if (dataPoints) {
                this.setTimeSeries(i,dataKey, dataTitle, dataPoints, dataUnit, stepped)
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
        if (this.chart) {
            this.chart.destroy();
            this.chart = null;
        }
        // let chartCanvas = this.getElementById('_myChart').getContext("2d");
        // let chartCanvas = shadowRoot.querySelector("[canvas]").getContext("2d");
        this.config.type = this.chartType
        this.config.options.plugins.title = {
            display: (!!this.chartTitle),
            text:  this.chartTitle
        }
        this.chart = new Chart(this.chartCanvas, this.config);
    }

    // experiment inject a value
    // if time is empty then use 'now'
    // key is the dataset key to add to. Default is the first one.
    addValue = (time, val, key) => {
        if (!time) {
            time = luxon.DateTime.now().toISO()
        }
        let ds = this.config.data.datasets[0];
        // locate the dataset
        for (let i = 0; i < this.config.data.datasets.length; i++) {
            let hasKey = this.config.data.datasets[i].key === key;
            if (hasKey) {
                ds = this.config.data.datasets[i];
                break
            }
        }

        ds.data.unshift({x:time,y:val})

        let endTime = luxon.DateTime.fromISO(time)
        if (endTime > this.timestamp) {
            this.timestamp = endTime
        }
        let newStartTime = endTime.plus((this.duration-1800)*1000)
        let newEndTime = endTime.plus({minute:30})

        // modify the start time so its exactly 24 hours before the end time
        this.config.options.scales.x.min = newStartTime.toISO();
        this.config.options.scales.x.max = newEndTime.toISO();


        this.chart.update();
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
    setTimeSeries = (nr,key, label, timePoints, dataUnit, stepped) => {
        const colors = ["#1e81b0", "#e28743", "#459b44", "#d1c684"]

        let yaxisID = dataUnit ? dataUnit : "default"
        // construct a replacement dataset
        console.log("setTimeSeries", timePoints.length, "items", "label=",label)
        let dataset = {
            key:key,
            // label: label,
            data: timePoints,
            borderWidth: 1,
            borderColor: colors[nr],
            fill: false,
            label: label + " " + dataUnit,
            // stepped: stepped?'after':false,
            stepped: 'after',
            tension: stepped? 0 : 0.1,  // bezier curve tension
            yAxisID: yaxisID
        }
        // Setup the y-axis scale for this dataset
        // Scales are based on the data unit. Add a scale if it doesnt exist.
        let hasScale= this.config.options.scales[yaxisID]
        if (!hasScale) {
            // the first scale is at the left, additional scales are on the right
            let isFirstScale = (nr === 0);
            this.config.options.scales[yaxisID] = {
                // backgroundColor: 'green',
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
                    color: colors[nr],
                    callback: (val, index, ticks) => {
                        return val + " " + dataUnit
                    }
                }
            }
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

