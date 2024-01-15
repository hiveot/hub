/**
 * Modal web-component
 * Features:
 * - open by setting show attribute on this modal element
 * - close by:
 *   1. removing show attribute from this element
 *   2. Press the ESC key
 *   3. click the close icon in the top right, if enabled
 *   4. click the close button in the footer, if enabled
 *   5. click outside the modal
 *   Close should be treated as cancel.
 *
 * - show a close icon in the top right corner with the closeIcon attribute
 * - show a close button in the footer with the 'closeButton' attribute (when not using okCancel)
 * - show cancel and accept buttons in the footer when the 'okCancelButton' attribute is set
 * - slide in and out when 'animate' attribute is set. (todo)
 * - placement: center, top, bottom, left, right (todo)
 *
 * Custom attributes:
 * - showclose  show the close icon button in the topright modal corner
 */
// const template = document.createElement('template')
// template.innerHTML = `
const template = `
<div _="on mutation of anything log 'element event'"
_="on event log 'element event'">
<slot></slot>
</div>
`


class HModal extends HTMLElement {

    constructor() {
        super();
        // this.attachShadow({mode: "open"});
        // this.shadowRoot.append(template.content.cloneNode(true));
        // this.shadowRoot.innerHTML = template;
        this.innerHTML = template
        let mo = new window.MutationObserver(
            function (arg) {
                // console.log("mutation observed", arg)
            })
        mo.observe(this, {childList: true, subtree: false})
        // this.querySelector()
    }

    attributeChangedCallback(name, oldValue, newValue) {
        console.log("h-modal3 attributeChangedCallback: " + name + "=" + newValue);
    }

    adoptedCallback() {
        console.log("h-modal3 adoptedCallback")
    }

    connectedCallback() {
        console.log("h-modal3 connectedCallback")
    }


    disconnectedCallback() {
        console.log("h-modal3 disconnectedCallback")
    }


    // supports slots in light DOM
    // https://frontendmasters.com/blog/light-dom-only/
    childrenToSlots(html, src_elems) {
        var template = document.createElement("template");
        template.innerHTML = html;

        const slots = template.content.querySelectorAll("slot");
        for (const slot of slots) {
            const slotChildren = this.querySelectorAll(`[slot='${slot.name}']`);
            slot.replaceWith(...slotChildren);
        }

        this.replaceChildren(template.content);
    }

}


customElements.define('h-modal3', HModal)
