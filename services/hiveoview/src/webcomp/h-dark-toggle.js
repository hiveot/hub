/* h-dark-toggle
 * This is a simple web element that toggles the pico.css "data-theme" attribute on the body
 * between 'light' and 'dark'.
 */
const themeAttrName = "data-theme"

const template = `
    <button target="_blank" dark-toggle-button  
            class="h-icon-button outline">
            <iconify-icon icon="mdi:theme-light-dark"></iconify-icon>
    </button>

    <style>

    .h-dark-toggle-img {
        height:20px;
    }

    </style>
`;

class HDarkToggle extends HTMLElement {

    static get observedAttributes() {
    }

    static setThemeFromLocalStorage() {
        let currentTheme = localStorage.getItem(themeAttrName)
        HDarkToggle.setTheme(currentTheme)
    }

    // set the theme and persist in local storage
    // if name is "" then remove the current theme in favor of the default
    static setTheme(name) {
        if (name == "") {
            document.documentElement.removeAttribute(themeAttrName)
        } else {
            document.documentElement.setAttribute(themeAttrName, name)
            localStorage.setItem(themeAttrName, name);
        }
    }

    constructor() {
        super();
        HDarkToggle.setThemeFromLocalStorage();
        this.innerHTML = template;
    }

    attributeChangedCallback(name, oldValue, newValue) {
        // console.log("attributeChangedCallback: " + name + "=" + newValue);
    }

    connectedCallback() {
        this.toggleButtonEl = this.querySelector("[dark-toggle-button]")
        this.toggleButtonEl.addEventListener("click", this.toggleDarkMode.bind(this))
    }

    toggleDarkMode(ev) {
        // console.log("toggleDarkMode");
        let currentTheme = document.documentElement.getAttribute(themeAttrName);
        if (currentTheme == "dark") {
            HDarkToggle.setTheme("light")
        } else {
            HDarkToggle.setTheme("dark")
        }
        if (ev) {
            ev.stopImmediatePropagation();
        }
    }

}

window.HDarkToggle = HDarkToggle
customElements.define('h-dark-toggle', HDarkToggle)

