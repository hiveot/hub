/**
 * Component managing a group of details sections. This:
 * 1. On first render set the name of all 'details' elements to the group name.
 *    this will make them behave as an according where only one element is open.
 * 2. On first render open the last saved 'details' element
 * 3. On opening a different section, save the element into session storage
 */

class HDetailsGroup extends HTMLElement {

    constructor() {
        super();
        this.groupName = ""
        // this.innerHTML = template;
    }

    connectedCallback() {
        let eList = this.querySelectorAll(
            ["details"])

        eList.forEach((item) => {
            //console.log("found item: ",item.id)
            item.name = this.groupName
            item.addEventListener("toggle", this.onToggle.bind(this))
        })
        this.openSavedDetails()
    }

    attributeChangedCallback(name, oldValue, newValue) {
        // console.log("h-modal attributeChangedCallback: " + name + "=" + newValue);
        if (name === "group-name" || name === "groupName") {
            this.groupName = newValue
        }
    }

    static get observedAttributes() {
        return ["groupName", "group-name"];
    }

    // when toggling, open details. Save the open details section.
    onToggle(ev) {
        if (ev.newState === "open") {
            sessionStorage.setItem(this.groupName, ev.currentTarget.id);
        } else {
            // When switching to another details section the first event is open followed
            // by close of the old one.
            // Only remove the opened item from storage if the close was intentional
            // rather than the result of opening another.
            let elementID = sessionStorage.getItem(this.groupName);
            if (ev.currentTarget.id == elementID) {
                // close was intentional
                sessionStorage.removeItem(this.groupName);
            }
        }
    }
    openSavedDetails() {
        let elementID = sessionStorage.getItem(this.groupName);
        if (elementID) {
            let detailsSection = document.getElementById(elementID);
            if (detailsSection) {
                detailsSection.open = true;
            }
        }
    }
}
customElements.define('h-details-group', HDetailsGroup)
