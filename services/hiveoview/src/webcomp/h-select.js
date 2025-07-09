// Select Webcomponent to support dynamic selection using a 'value' attribute
//

// let baseClass = document.createElement('ul').constructor
// const HList = class extends baseClass {
class HSelect extends HTMLSelectElement {

    constructor() {
        super();
        this.value = ""
    }

    static get observedAttributes() {
        return ["value"];
    }
    attributeChangedCallback(name, oldValue, newValue) {
        if (name === "value") {
            // console.log("Set value to:",newValue)
            this.value = newValue
            this.updateSelection(newValue)
        }
    }

    // Set selected on the option with a value of the given name
    updateSelection(name) {}

}
customElements.define('h-select', HSelect, { extends: "select"});
