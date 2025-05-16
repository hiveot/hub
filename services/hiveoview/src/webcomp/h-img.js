// Web element with img supporting refresh interval
const template = `
<img alt="" src="" class="h-img">
   
<style>
    .h-img {
    width:100%;
    height:100%;
    object-fit:cover;
    }
</style>
`

/**
 @param src = URL source with variable support:
 @param interval = seconds between polling
 */
class HImg extends HTMLElement {

    constructor() {
        super();
        this.innerHTML = template;
        this.imgEl = this.getElementsByTagName("img")[0]
        this.intervalSec = ""
        this.src = ""
    }

    attributeChangedCallback(name, oldValue, newValue) {
        // console.log("h-modal attributeChangedCallback: " + name + "=" + newValue);
        if (name === "src") {
            this.src = newValue
        } else if (name === "interval") {
            this.intervalSec = newValue
        }
    }

    static get observedAttributes() {
        return ["src", "interval"];
    }

    connectedCallback() {
        this.imgEl.src= this.src
        if (this.intervalSec && this.intervalSec > 3) {
            // console.log("h-img interval=",this.intervalSec)

            this.intervalID = setInterval(this.reloadImage.bind(this), this.intervalSec*1000)
        }
    }
    disconnectedCallback() {
        if (this.intervalID) {
            clearInterval(this.intervalID)
        }
    }
    reloadImage() {
        let ts = new Date().getTime().toString()
        let newSrc = this.src + "?&ts="+ts
        console.log("reloading image: src=",newSrc)
        this.imgEl.src = newSrc
    }
}

customElements.define('h-img', HImg)
