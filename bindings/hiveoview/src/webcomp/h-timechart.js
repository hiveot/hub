// This is a simple time series chart component wrapper around chartjs.
//
// Intended to easily present sensor data as a line, area or bar graph without
// having to wade through the expansive chartjs documentation.
//
// The default graph type is line, but setting a data series can overwrite this
//
//
// properties:
//  graphType: line,
//  title: brand title text
//

// import  '../static/chartjs-4.4.3.js'
import  '../static/chartjs-4.4.1.umd.js'
import  '../static/chartjs-adapter-date-fns.bundle-v3.0.0.min.js'

// https://www.chartjs.org/docs/latest/configuration/responsive.html
const template = `
    <div  style="position:relative; width:100%; height:60vh" 
    >
        <canvas id="_myChart" style="position:absolute" ></canvas>
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
            // labels: ['Red', 'Blue', 'Yellow', 'Green', 'Purple', 'Orange'],
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
            // IMPORTANT: in order to resize properly, set responsive to true
            // and disable maintainAspectRatio. Place the canvas in a div with
            // position:relative and set size on the div.
            // https://www.chartjs.org/docs/latest/configuration/responsive.html#configuration-options
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                x: {
                    grid: {
                        drawTicks: true
                    },
                    border: {
                        display:true,
                        width:2,
                        dash: [2]
                    },
                    ticks: {
                        major:true,
                        maxTicksLimit: 24,
                    },
                    type: 'time',
                    time: {
                        unit: 'minute',
                        tooltipFormat: 'yyyy-MM-dd HH:mm',
                        displayFormats: {
                            second: 'HH:mm:ss',
                            minute: 'HH:mm',
                            hour: 'HH',
                        }
                    }
                },
                y: {
                    beginAtZero: true,
                    grid: {
                        // test to highlight the row of a certain value
                        color: (ctx)=> {
                            return (ctx.tick.value === this.highlightYrow)?"green":'rgba(0,0,0,0.1)'
                        }
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
        return ['dataset']
    }


    constructor() {
        super();
        // HChartJS.setThemeFromLocalStorage();
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
        }
    }

    attributeChangedCallback(name, oldValue, newValue) {
    }

    render = () => {
        if (this.chart) {
            this.chart.destroy();
            this.chart = null;
        }
        let chartCanvas = document.getElementById('_myChart').getContext("2d");
        this.chart = new Chart(chartCanvas, this.config);
    }

    // Insert or replace a chartjs dataset at the given index
    //
    // @param nr is the index to insert it at
    // @param ds is the array of [{x:timestamp,y:value},...]
    setDataSet = (nr, ds) => {
        this.config.data.datasets[nr] = ds;
    }

    // Set a new time series to display.
    // This is a simple helper function for common use-case.
    //
    // @param nr is the series index, 0 for default, 1... for multiple series
    // @param label is the label of this series
    // @param timePoints is an array of: {x:timestamp, y:value}
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
        this.chart.update();
    }

}

window.HChartJS = HTimechart
customElements.define('h-timechart', HTimechart)

