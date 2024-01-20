/**
 * TabBar web-component using the light DOM
 *
 * Features:
 * - Show a navigation bar with options to select from
 *
 * Element attributes:
 *
 * Element styles:
 *  .show@s   show on small media
 *  .show@m   show on medium media
 *  .hide@s   hide on small media
 *  .hide@m   hide on medium media
 *  .tabs     show nav items as a set of tabs tab
 */

const template = document.createElement('template')
template.innerHTML = `
<nav>  
    <ul class="navbar">
        <slot>navbar empty slot content</slot>
    </ul>
</nav>

<style>
  
 .navbar {
    display: flex;  /* hide the list bullets */
    flex-direction: row; 
  padding-left: 10px;
  padding-right: 10px;
    margin:0px;
    border 1px solid gray;
  }
.navbar li {
}
  

</style>
`

/**
 * # h-navbar custom web component
 * Usage:
 *   <h-navbar>
 *     <a href="#item1">label 1</div>
 *     <a href="#item2">label 2</div>
 *  </h-navbar>
 *
 * @attribute show: show the tabbar on initial render
 */
class HNavbar extends HTMLElement {

    constructor() {
        super();
        // shadowroot lets the slot content be replaced using hx-get, without it the modal handling div's would be replaced.
        const shadowRoot = this.attachShadow({mode: "open"});
        // clone the template to support multiple instances
        shadowRoot.append(template.content.cloneNode(true));
        // this.innerHTML = template.innerHTML

    }

    static get observedAttributes() {
        return ["show"];
    }

    attributeChangedCallback(name, oldValue, newValue) {
        // console.log("h-modal attributeChangedCallback: " + name + "=" + newValue);
        if (name === "show") {
            // handled through css
        } else if (name === "shadow") {
            // handled through css
        }
    }

    adoptedCallback() {
        // console.log("h-modal adoptedCallback")
    }

    connectedCallback() {
        // console.log("h-modal connectedCallback")
    }

    closeModal() {
        this.removeAttribute("show")
    }

    disconnectedCallback() {
        // console.log("h-modal disconnectedCallback")
    }


}


customElements.define('h-navbar', HNavbar)
