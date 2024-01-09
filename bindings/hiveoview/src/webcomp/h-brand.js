// This is a simple webcomponent for the purpose of learning how to build one.
// Issues:
//  1. How to import the html from an external file?
//  2. Is it possible to define HTML, JS, and CSS in a single file like vue does?
//  3. How to test this easier without a nodejs environment (eg golang)?
//  4. Can this component be styled from the outside?  eg: h-brand {} => use ::part(name){}
//  5. Should this use :host ?  not sure
//  6. How to document the properties of this component?
//
// properties:
//  logo:  path to image file
//  title: brand title text
//
const template = document.createElement('template')
template.innerHTML = `
    <div class="h-brand">
        <slot name="logo"><img logo  part="h-brand-logo"/></slot>
        <slot name="title"><text title part="h-brand-title"></texttitle></slot>
        <slot></slot>
    </div>

    <style>
        /* @import url("/path/to/common/stylesheet.css"); */
          .h-brand {
            display: flex;
            height: 90%;
            flex-direction: row;
            flex-wrap: nowrap;
            align-items: center;
            
            /* nesting not supported in older android browsers*/ 
            /*img {*/
            /*    !*tame the SVG icons size*!*/
            /*    height: 46px;*/
            /*}*/
        }
        .h-brand  img {
            /*tame the SVG icons size*/
            height: 46px;
        }
        :host {
        /* applies to shadow root (container of this component) itself*/
        /* by default web components are display inline (no dimension) */
        /*    display: block;*/
        }
        ::slotted(img) {
            /*max-height:62px !important;*/
        }
        ::part() {
        }
    
</style>
`;


class HBrand extends HTMLElement {

    constructor() {
        super();
        const shadowRoot = this.attachShadow({mode: "open"});
        shadowRoot.append(template.content.cloneNode(true));
        this.elLogo = shadowRoot.querySelector("[logo]");
        this.elTitle = shadowRoot.querySelector("[title]");
        if (this.elTitle == null) {
            console.error("title selector not found");
        }
    }

    static get observedAttributes() {
        return ["logo", "title"];
    }

    attributeChangedCallback(name, oldValue, newValue) {
        // console.log("attributeChangedCallback: " + name + "=" + newValue);
        // let logoSelector = shadowRoot.querySelector("[logo]")
        if (name === "logo") {
            this.elLogo.src = newValue;
        } else if (name === "title") {
            // this.elTitle.innerHTML = newValue
            // this.elTitle.innerText = newValue
            this.elTitle.textContent = newValue;
        }
        // console.log("height:" + this.style.getPropertyValue("height"));
    }


    connectedCallback() {
    }

}

customElements.define('h-brand', HBrand)

