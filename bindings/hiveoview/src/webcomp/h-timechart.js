// This is a simple time series chart component wrapper around chartjs.
//
// Intended to easily present sensor data as a line, area or bar graph without
// having to wade through the expansive chartjs documentation.
//
// The default graph type is line, but setting a data series can overwrite this
//
//
// properties:
//  chart-type: line
//  chart-title: brand title text
//

// Note: chartjs requires import of a date library. date-fns or luxon will do
// import  '../static/chartjs-4.4.3.js'
// import  '../static/chartjs-4.4.1.umd.js'
// import  '../static/chartjs-adapter-date-fns.bundle-v3.0.0.min.js'

export const PropChartType = "chart-type"
export const PropChartTitle = "chart-title"
// FIXME: generate random ID for canvas for multiple instances
// https://www.chartjs.org/docs/latest/configuration/responsive.html
const template = `
    <div  style="position:relative; width:100%; height:60vh" 
    >
        <canvas id="_myChart" style="position:absolute; padding: 5px; margin:5px" ></canvas>
    </div>  
`

export class HTimechart extends HTMLElement {
    // value to highlight the y-axis row
    highlightYrow = 20

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
            layout: {
              padding: {
                  right: 20
              }
              //   autopadding:false
            },
            animation: {
                duration: 500
            },
            // spanGaps: 1000*60*60*5,
            // IMPORTANT: in order to resize properly, set responsive to true
            // and disable maintainAspectRatio. Place the canvas in a div with
            // position:relative and set size on the div.
            // https://www.chartjs.org/docs/latest/configuration/responsive.html#configuration-options
            responsive: true,
            maintainAspectRatio: false,
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
                y: {
                    // min: -20,
                    // max: 40,
                    // beginAtZero: true,
                    grid: {
                        // test to highlight the row of a certain value
                        color: (ctx)=> {
                            return (ctx.tick.value === this.highlightYrow)?"green":'rgba(0,0,0,0.1)'
                        }
                    },
                    ticks: {
                        maxTicksLimit: 30,
                        source: "auto"
                    }
                }
            },
            plugins: {
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
        return [ PropChartType, PropChartTitle]
    }


    constructor() {
        super();
        this.chartType = 'line'
        this.chartTitle = undefined
        this.innerHTML = template;
    }

    connectedCallback() {
        let readyEvent = new Event('on-timechart-ready', {
            bubbles:true,
            cancelable: false,
            composed: true
        })
        this.render();
        this.dispatchEvent(readyEvent);
    }

    disconnectedCallback() {
        if (this.chart) {
            this.chart.destroy();
            this.chart = null
        }
    }

    attributeChangedCallback(name, oldValue, newValue) {
        if (name === PropChartTitle) {
            this.chartTitle = newValue
        } else if (name === PropChartType) {
            this.chartType = newValue
        }
    }

    render = () => {
        if (this.chart) {
            this.chart.destroy();
            this.chart = null;
        }
        let chartCanvas = document.getElementById('_myChart').getContext("2d");
        this.config.type = this.chartType
        this.config.options.plugins.title = {
            display: (!!this.chartTitle),
            text:  this.chartTitle
        }
        this.chart = new Chart(chartCanvas, this.config);
    }

    // experiment inject a value
    // if time is empty then use 'now'
    addValue = (time, val) => {
        if (!time) {
            time = luxon.DateTime.now().toISO()
        }

        let ds = this.config.data.datasets[0].data;
        ds.unshift({x:time,y:val})


        let endTime = luxon.DateTime.fromISO(time)
        let newStartTime = endTime.plus({hours: -24, minutes:-30}).toISO()
        let newEndTime = endTime.plus({minute:30}).toISO()

        // modify the start time so its exactly 24 hours before the end time
        this.config.options.scales.x.min = newStartTime;
        this.config.options.scales.x.max = newEndTime;


        this.chart.update();
    }

    // Insert or replace a chartjs dataset at the given index
    //
    // @param nr is the index to insert it at
    // @param ds is the array of [{x:timestamp,y:value},...]
    setDataSet = (nr, ds) => {
        // NOTE1: horrible hack to show the last vertical grid line at the last timestamp
        // NOTE2: this expects the time range to be newest to oldest, the further in the older it gets
        // FIXME: the intended end time is needed in case of missing data
        this.config.data.datasets[nr] = ds;
        // let startItem = ds.data.at(-1)
        // let startTime = luxon.DateTime.fromISO(startItem["x"])
        let endItem = ds.data.at(0)
        let endTime = luxon.DateTime.fromISO(endItem["x"])

        let newStartTime = endTime.plus({hours: -24, minutes:-30}).toISO()
        let newEndTime = luxon.DateTime.fromISO(endItem["x"]).plus({minute:30}).toISO()

        // modify the start time so its exactly 24 hours before the end time
        this.config.options.scales.x.min = newStartTime;
        this.config.options.scales.x.max = newEndTime;

        this.chart.update();
    }

    // Set a new time series to display.
    // This is a simple helper function for common use-case.
    //
    // @param nr is the series index, 0 for default, 1... for multiple series
    // @param label is the label of this series
    // @param timePoints is an array of: {x:timestamp, y:value} in reverse order (newest first)
    setTimeSeries = (nr, label, timePoints) => {
        // construct a replacement dataset
        let ds = {
            // label: label,
            data: timePoints,
            borderWidth: 1
        }
        if (label) {
            ds.label = label;
        }
        this.setDataSet(nr, ds);
    }

}

// window.HChartJS = HTimechart
customElements.define('h-timechart', HTimechart)

